/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package p2p

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	kcp "github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/nacl/box"

	"github.com/mysteriumnetwork/node/router"
	"github.com/mysteriumnetwork/node/trace"
)

var (
	// If this env variable is set channel will log raw messages from send and receive loops.
	debugTransport = os.Getenv("P2P_DEBUG_TRANSPORT") == "1"

	// ErrSendTimeout indicates send timeout error.
	ErrSendTimeout = errors.New("p2p send timeout")

	// ErrHandlerNotFound indicates that peer is not registered handler yet.
	ErrHandlerNotFound = errors.New("p2p peer handler not found")
)

const (
	kcpMTUSize            = 1280
	mtuLimit              = 1500
	initialTrafficTimeout = 30 * time.Second
)

// ChannelSender is used to send messages.
type ChannelSender interface {
	// Send sends message to given topic. Peer listening to topic will receive message.
	Send(ctx context.Context, topic string, msg *Message) (*Message, error)
}

// ChannelHandler is used to handle messages.
type ChannelHandler interface {
	// Handle registers handler for given topic which handles peer request.
	Handle(topic string, handler HandlerFunc)
}

// Channel represents p2p communication channel which can send and receive messages over encrypted and reliable UDP transport.
type Channel interface {
	ChannelSender
	ChannelHandler

	// Tracer returns tracer which tracks channel establishment
	Tracer() *trace.Tracer

	// ServiceConn returns UDP connection which can be used for services.
	ServiceConn() *net.UDPConn

	// Conn returns underlying channel's UDP connection.
	Conn() *net.UDPConn

	// Close closes p2p communication channel.
	Close() error
}

// HandlerFunc is channel request handler func signature.
type HandlerFunc func(c Context) error

// stream is used to associate request and reply messages.
type stream struct {
	id    uint64
	resCh chan *transportMsg
}

type peer struct {
	sync.RWMutex
	publicKey  PublicKey
	remoteAddr *net.UDPAddr
}

func (p *peer) addr() *net.UDPAddr {
	p.RLock()
	defer p.RUnlock()

	return p.remoteAddr
}

func (p *peer) updateAddr(addr *net.UDPAddr) {
	p.Lock()
	defer p.Unlock()

	p.remoteAddr = addr
}

// transport wraps network primitives for sending and receiving packets.
type transport struct {
	// textReader is used to read p2p text protocol data.
	textReader *textproto.Reader

	// textWriter is used to marshal messages to underlying text protocol.
	textWriter *textproto.Writer

	// session is KCP session which wraps UDP connection and adds reliability and ordered messages support.
	session *kcp.UDPSession

	// remoteConn is initial conn which should be created from NAT hole punching or manually. It contains
	// initial local and remote peer addresses.
	remoteConn *net.UDPConn

	// proxyConn is used for KCP session as a remote. Since KCP doesn't expose it's data read loop
	// this is needed to detect remote peer address changes as we can simply use conn.ReadFromUDP and
	// get updated peer address.
	proxyConn *net.UDPConn
}

// channel implements Channel interface.
type channel struct {
	mu   sync.RWMutex
	once sync.Once

	// tr is transport containing network related connections for p2p to work.
	tr *transport

	tracer *trace.Tracer

	// serviceConn is separate connection which is created outside of p2p channel when
	// performing initial NAT hole punching or manual conn. It is here just because it's more easy
	// to pass it to services as p2p channel will be available anyway.
	serviceConn *net.UDPConn

	// topicHandlers is similar to HTTP Server handlers and is responsible for handling peer requests.
	topicHandlers map[string]HandlerFunc

	// streams is temp map to create request/response pipelines. Each stream is created on send and contains
	// channel to which receive loop should eventually send peer reply.
	streams      map[uint64]*stream
	nextStreamID uint64

	// privateKey is channel's private key. For now it's here just to be able to recreate the same channel for unit tests.
	privateKey PrivateKey

	// peer is remote peer holding it's public key and address.
	peer *peer

	// localSessionAddr is KCP UDP conn address to which packets are written from remote conn.
	localSessionAddr *net.UDPAddr

	// sendQueue is a queue to which channel puts messages for sending. Message is not send directly to remote peer
	// but to proxy conn which is when responsible for sending to remote
	sendQueue chan *transportMsg

	// upnpPortsRelease should be called to close mapped upnp ports when channel is closed.
	upnpPortsRelease []func()

	// stop is used to stop all running goroutines.
	stop chan struct{}

	// weather channel saw remote traffic
	remoteAlive chan struct{}

	// terminate remote aliveness checking only once
	remoteAliveOnce sync.Once
}

// newChannel creates new p2p channel with initialized crypto primitives for data encryption
// and starts listening for connections.
func newChannel(remoteConn *net.UDPConn, privateKey PrivateKey, peerPubKey PublicKey) (*channel, error) {
	peerAddr := remoteConn.RemoteAddr().(*net.UDPAddr)
	localAddr := remoteConn.LocalAddr().(*net.UDPAddr)
	remoteConn, err := reopenConn(remoteConn)
	if err != nil {
		return nil, fmt.Errorf("could not reopen remote conn: %w", err)
	}

	// Create local proxy UDP conn which will receive packets from local KCP UDP conn.
	proxyConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		return nil, fmt.Errorf("could not create proxy conn: %w", err)
	}

	// Setup KCP session. It will write to proxy conn only.
	udpSession, sessAddr, err := listenUDPSession(proxyConn.LocalAddr(), privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not create KCP UDP session: %w", err)
	}

	log.Debug().Msgf("Creating p2p channel with local addr: %s, UDP session addr: %s, proxy addr: %s, remote peer addr: x.x.x.x:%d", localAddr.String(), udpSession.LocalAddr().String(), proxyConn.LocalAddr().String(), peerAddr.Port)

	tr := transport{
		textReader: textproto.NewReader(bufio.NewReader(udpSession)),
		textWriter: textproto.NewWriter(bufio.NewWriter(udpSession)),
		session:    udpSession,
		remoteConn: remoteConn,
		proxyConn:  proxyConn,
	}

	peer := peer{
		publicKey:  peerPubKey,
		remoteAddr: peerAddr,
	}

	c := channel{
		tr:               &tr,
		topicHandlers:    make(map[string]HandlerFunc),
		streams:          make(map[uint64]*stream),
		privateKey:       privateKey,
		peer:             &peer,
		localSessionAddr: sessAddr,
		serviceConn:      nil,
		stop:             make(chan struct{}, 1),
		sendQueue:        make(chan *transportMsg, 100),
		remoteAlive:      make(chan struct{}, 1),
	}

	return &c, nil
}

func (c *channel) launchReadSendLoops() {
	go c.remoteReadLoop()
	go c.remoteSendLoop()
	go c.localReadLoop()
	go c.localSendLoop()
}

// remoteReadLoop reads from remote conn and writes to local KCP UDP conn.
// If remote peer addr changes it will be updated and next send will use new addr.
func (c *channel) remoteReadLoop() {
	buf := make([]byte, mtuLimit)
	latestPeerAddr := c.peer.addr()

	go c.checkIfChannelAlive()

	for {
		select {
		case <-c.stop:
			return
		default:
		}

		n, addr, err := c.tr.remoteConn.ReadFrom(buf)
		if err != nil {
			if !errNetClose(err) {
				log.Error().Err(err).Msg("Read from remote conn failed")
			}
			return
		}

		c.remoteAliveOnce.Do(func() {
			close(c.remoteAlive)
		})

		// Check if peer address changed.
		if addr, ok := addr.(*net.UDPAddr); ok {
			if !addr.IP.Equal(latestPeerAddr.IP) || addr.Port != latestPeerAddr.Port {
				log.Debug().Msgf("Peer address changed from %v to %v", latestPeerAddr, addr)
				c.peer.updateAddr(addr)
				latestPeerAddr = addr
			}
		}

		_, err = c.tr.proxyConn.WriteToUDP(buf[:n], c.localSessionAddr)
		if err != nil {
			if !errNetClose(err) {
				log.Error().Err(err).Msg("Write to local udp session failed")
			}
			return
		}
	}
}

// remoteSendLoop reads from proxy conn and writes to remote conn.
// Packets to proxy conn are written by local KCP UDP session from localSendLoop.
func (c *channel) remoteSendLoop() {
	buf := make([]byte, mtuLimit)
	for {
		select {
		case <-c.stop:
			return
		default:
		}

		n, err := c.tr.proxyConn.Read(buf)
		if err != nil {
			if !errNetClose(err) {
				log.Error().Err(err).Msg("Read from proxy conn failed")
			}
			return
		}

		_, err = c.tr.remoteConn.WriteToUDP(buf[:n], c.peer.addr())
		if err != nil {
			if !errNetClose(err) {
				log.Error().Err(err).Msgf("Write to remote peer conn failed")
			}
			return
		}
	}
}

// localReadLoop reads incoming requests or replies to initiated requests.
func (c *channel) localReadLoop() {
	for {
		select {
		case <-c.stop:
			return
		default:
		}

		var msg transportMsg
		if err := msg.readFrom(c.tr.textReader); err != nil {
			if !errPipeClosed(err) && !errNetClose(err) {
				log.Err(err).Msg("Read from textproto reader failed")
			}
			return
		}

		if debugTransport {
			fmt.Printf("recv from %s: %+v\n", c.tr.session.RemoteAddr(), msg)
		}

		// If message contains topic it means that peer is making a request
		// and waits for response.
		if msg.topic != "" {
			go c.handleRequest(&msg)
		} else {
			// In other case we treat it as a reply for peer to our request.
			go c.handleReply(&msg)
		}
	}
}

// localSendLoop sends data to local proxy conn.
func (c *channel) localSendLoop() {
	for {
		select {
		case <-c.stop:
			return
		case msg, more := <-c.sendQueue:
			if !more {
				return
			}

			if debugTransport {
				fmt.Printf("send to %s: %+v\n", c.tr.session.RemoteAddr(), msg)
			}

			if err := msg.writeTo(c.tr.textWriter); err != nil {
				if !errPipeClosed(err) && !errNetClose(err) {
					log.Err(err).Msg("Write to textproto writer failed")
				}
				return
			}
		}
	}
}

// handleReply forwards reply message to associated stream result channel.
func (c *channel) handleReply(msg *transportMsg) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.streams[msg.id]; ok {
		s.resCh <- msg
	} else {
		log.Warn().Msgf("Stream %d not found, message data: %s", msg.id, string(msg.data))
	}
}

// handleRequest handles incoming request and schedules reply to send queue.
func (c *channel) handleRequest(msg *transportMsg) {
	c.mu.RLock()
	handler, ok := c.topicHandlers[msg.topic]
	c.mu.RUnlock()

	var resMsg transportMsg
	resMsg.id = msg.id

	if !ok {
		resMsg.statusCode = statusCodeHandlerNotFoundErr
		errMsg := fmt.Sprintf("handler %q not found", msg.topic)
		log.Err(errors.New(errMsg))
		resMsg.data = []byte(errMsg)
		c.sendQueue <- &resMsg
		return
	}

	ctx := defaultContext{req: &Message{Data: msg.data}}
	err := handler(&ctx)
	if err != nil {
		log.Err(err).Msgf("Handler %q internal error", msg.topic)
		resMsg.statusCode = statusCodeInternalErr
		resMsg.msg = err.Error()
	} else if ctx.publicError != nil {
		log.Err(ctx.publicError).Msgf("Handler %q public error", msg.topic)
		resMsg.statusCode = statusCodePublicErr
		resMsg.data = []byte(ctx.publicError.Error())
	} else {
		resMsg.statusCode = statusCodeOK
		if ctx.res != nil {
			resMsg.data = ctx.res.Data
		}
	}
	c.sendQueue <- &resMsg
}

// Tracer returns tracer which tracks channel establishment
func (c *channel) Tracer() *trace.Tracer {
	return c.tracer
}

// ServiceConn returns UDP connection which can be used for services.
func (c *channel) ServiceConn() *net.UDPConn {
	return c.serviceConn
}

// Close closes channel.
func (c *channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var closeErr error
	c.once.Do(func() {
		close(c.stop)
		for _, release := range c.upnpPortsRelease {
			release()
		}

		if err := c.tr.remoteConn.Close(); err != nil {
			closeErr = fmt.Errorf("could not close remote conn: %w", err)
		}

		if err := c.tr.proxyConn.Close(); err != nil {
			closeErr = fmt.Errorf("could not close proxy conn: %w", err)
		}

		if err := c.tr.session.Close(); err != nil {
			closeErr = fmt.Errorf("could not close p2p transport session: %w", err)
		}

		if c.serviceConn != nil {
			if err := c.serviceConn.Close(); err != nil {
				if errors.Is(err, errors.New("use of closed network connection")) { // Have to check this error as a string match https://github.com/golang/go/issues/4373
					closeErr = fmt.Errorf("could not close p2p service connection: %w", err)
				}
			}
		}
	})

	return closeErr
}

// Conn returns underlying channel's UDP connection.
func (c *channel) Conn() *net.UDPConn {
	return c.tr.remoteConn
}

// Send sends message to given topic. Peer listening to topic will receive message.
func (c *channel) Send(ctx context.Context, topic string, msg *Message) (*Message, error) {
	reply, err := c.sendRequest(ctx, topic, msg)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// Handle registers handler for given topic which handles peer request.
func (c *channel) Handle(topic string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.topicHandlers[topic] = handler
}

// sendRequest sends message to send queue and waits for response.
func (c *channel) sendRequest(ctx context.Context, topic string, m *Message) (*Message, error) {
	s := c.addStream()
	defer c.deleteStream(s.id)

	// Send request.
	c.sendQueue <- &transportMsg{id: s.id, topic: topic, data: m.Data}

	// Wait for response.
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for reply to %q: %w", topic, ErrSendTimeout)
	case res := <-s.resCh:
		if res.statusCode != statusCodeOK {
			if res.statusCode == statusCodePublicErr {
				return nil, fmt.Errorf("public peer error: %s", string(res.data))
			}
			if res.statusCode == statusCodeHandlerNotFoundErr {
				return nil, fmt.Errorf("%s: %w", string(res.data), ErrHandlerNotFound)
			}
			return nil, fmt.Errorf("peer error: %w", errors.New(res.msg))
		}
		return &Message{Data: res.data}, nil
	}
}

func (c *channel) addStream() *stream {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextStreamID++
	s := &stream{id: c.nextStreamID, resCh: make(chan *transportMsg, 1)}
	c.streams[s.id] = s
	return s
}

func (c *channel) deleteStream(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.streams, id)
}

func (c *channel) setTracer(tracer *trace.Tracer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tracer = tracer
}

func (c *channel) setServiceConn(conn *net.UDPConn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Debug().Msgf("Will use service conn with local port: %d, remote port: %d", conn.LocalAddr().(*net.UDPAddr).Port, conn.RemoteAddr().(*net.UDPAddr).Port)
	c.serviceConn = conn
}

func (c *channel) setUpnpPortsRelease(release []func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.upnpPortsRelease = release
}

func (c *channel) checkIfChannelAlive() {
	select {
	case <-c.stop:
	case <-c.remoteAlive:
		return
	case <-time.After(initialTrafficTimeout):
		log.Warn().Msgf("No initial traffic for %.0f sec. Terminating channel.", initialTrafficTimeout.Seconds())
		err := c.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close channel on inactivity")
		}
	}
}

func reopenConn(conn *net.UDPConn) (*net.UDPConn, error) {
	// conn first must be closed to prevent use of WriteTo with pre-connected connection error.
	conn.Close()
	conn, err := net.ListenUDP("udp4", conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		return nil, fmt.Errorf("could not listen UDP: %w", err)
	}

	if err := router.ProtectUDPConn(conn); err != nil {
		return nil, fmt.Errorf("failed to protect udp connection: %w", err)
	}

	return conn, nil
}

func listenUDPSession(proxyAddr net.Addr, privateKey PrivateKey, peerPubKey PublicKey) (sess *kcp.UDPSession, localAddr *net.UDPAddr, err error) {
	blockCrypt, err := newBlockCrypt(privateKey, peerPubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create block crypt: %w", err)
	}

	localConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create UDP conn: %w", err)
	}

	localAddr = localConn.LocalAddr().(*net.UDPAddr)

	sess, err = kcp.NewConn3(1, proxyAddr, blockCrypt, 10, 3, localConn)
	if err != nil {
		localConn.Close()
		return nil, nil, fmt.Errorf("could not create UDP session: %w", err)
	}

	sess.SetMtu(kcpMTUSize)

	return sess, localAddr, err
}

func newBlockCrypt(privateKey PrivateKey, peerPublicKey PublicKey) (kcp.BlockCrypt, error) {
	// Compute shared key. Nonce for each message will be added inside kcp salsa block crypt.
	var sharedKey [32]byte
	box.Precompute(&sharedKey, (*[32]byte)(&peerPublicKey), (*[32]byte)(&privateKey))
	blockCrypt, err := kcp.NewSalsa20BlockCrypt(sharedKey[:])
	if err != nil {
		return nil, fmt.Errorf("could not create Sasla20 block crypt: %w", err)
	}
	return blockCrypt, nil
}

func errNetClose(err error) bool {
	// Hack. See https://github.com/golang/go/issues/4373 which should expose net close error with 1.15.
	return strings.Contains(err.Error(), "use of closed network connection")
}

func errPipeClosed(err error) bool {
	// Hack. We can't check io.ErrPipeClosed as kcp wraps this error with old github.com/pkg/errors.
	return strings.Contains(err.Error(), "io: read/write on closed pipe")
}
