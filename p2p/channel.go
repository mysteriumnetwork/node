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
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/nacl/box"
)

// Channel represents p2p communication channel which can send and receive data over encrypted and reliable UDP transport.
type Channel interface {
	// Send sends data to given topic. Peer listening to topic will receive message.
	Send(topic string, msg *Message) (*Message, error)
	// Handle registers handler for given topic which handles peer request.
	Handle(topic string, handler HandlerFunc)
	// ServiceConn returns UDP connection which can be used for services.
	ServiceConn() *net.UDPConn
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

// channel implements Channel interface.
type channel struct {
	mu sync.RWMutex

	conn                  *textproto.Conn
	topicHandlers         map[string]HandlerFunc
	streams               map[uint64]*stream
	privateKey            PrivateKey
	blockCrypt            kcp.BlockCrypt
	sendTimeout           time.Duration
	serviceConn           *net.UDPConn
	keepAlivePingInterval time.Duration
	stop                  chan struct{}
	sendQueue             chan *transportMsg
}

// newChannel creates new p2p channel with initialized crypto primitives for data encryption
// and starts listening for connections.
func newChannel(rawConn *net.UDPConn, privateKey PrivateKey, peerPubKey PublicKey) (*channel, error) {
	blockCrypt, err := newBlockCrypt(privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not create block crypt: %w", err)
	}

	udpSession, err := dialUDPSession(rawConn, blockCrypt)
	if err != nil {
		return nil, fmt.Errorf("could not create UDP session: %w", err)
	}

	c := &channel{
		conn:                  textproto.NewConn(udpSession),
		topicHandlers:         make(map[string]HandlerFunc),
		streams:               make(map[uint64]*stream),
		privateKey:            privateKey,
		blockCrypt:            blockCrypt,
		sendTimeout:           30 * time.Second,
		keepAlivePingInterval: 2 * time.Second,
		serviceConn:           nil,
		stop:                  make(chan struct{}, 1),
		sendQueue:             make(chan *transportMsg, 100),
	}

	go c.readLoop()
	go c.sendLoop()
	go c.sendKeepaliveLoop()
	c.Handle(topicKeepAlive, func(c Context) error {
		log.Debug().Msg("Received P2P keep alive ping")
		return c.OK()
	})

	return c, nil
}

// readLoop reads incoming requests or replies to initiated requests.
func (c *channel) readLoop() {
	for {
		select {
		case <-c.stop:
			return
		default:
			var msg transportMsg
			if err := msg.readFrom(c.conn); err != nil {
				continue
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
}

// sendLoop sends data to underlying network.
func (c *channel) sendLoop() {
	for {
		select {
		case <-c.stop:
			return
		case msg, more := <-c.sendQueue:
			if !more {
				return
			}

			if err := msg.writeTo(c.conn); err != nil {
				log.Err(err).Msg("Write failed")
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
		resMsg.statusCode = statusCodePublicErr
		errMsg := fmt.Sprintf("handler %q is not registered", msg.topic)
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

// ServiceConn returns UDP connection which can be used for services.
func (c *channel) ServiceConn() *net.UDPConn {
	return c.serviceConn
}

// Close closes channel.
func (c *channel) Close() error {
	close(c.stop)
	return c.conn.Close()
}

// SetSendTimeout overrides default send timeout.
func (c *channel) SetSendTimeout(t time.Duration) {
	c.sendTimeout = t
}

// Send sends data to given topic. Peer listening to topic will receive message.
func (c *channel) Send(topic string, msg *Message) (*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.sendTimeout)
	defer cancel()

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

// Start keepalive ping send loop.
func (c *channel) sendKeepaliveLoop() {
	go func() {
		for {
			select {
			case <-c.stop:
				return
			case <-time.After(c.keepAlivePingInterval):
				if _, err := c.Send(topicKeepAlive, &Message{Data: []byte("PING")}); err != nil {
					log.Err(err).Msg("Failed to send P2P keepalive message")
				}
			}
		}
	}()
}

// sendRequest sends message to send queue and waits for response.
func (c *channel) sendRequest(ctx context.Context, topic string, m *Message) (*Message, error) {
	s := &stream{id: uint64(c.conn.Next()), resCh: make(chan *transportMsg, 1)}
	c.mu.Lock()
	c.streams[s.id] = s
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.streams, s.id)
		c.mu.Unlock()
	}()

	// Send request.
	c.sendQueue <- &transportMsg{id: s.id, topic: topic, data: m.Data}

	// Wait for response.
	select {
	case <-ctx.Done():
		return nil, errors.New("send timeout")
	case res := <-s.resCh:
		if res.statusCode != statusCodeOK {
			if res.statusCode == statusCodePublicErr {
				return nil, fmt.Errorf("public peer error: %s", string(res.data))
			}
			return nil, errors.New("internal peer error")
		}
		return &Message{Data: res.data}, nil
	}
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

func dialUDPSession(rawConn *net.UDPConn, block kcp.BlockCrypt) (*kcp.UDPSession, error) {
	rawConn.Close()

	raddr := rawConn.RemoteAddr().(*net.UDPAddr)
	network := "udp4"
	if raddr.IP.To4() == nil {
		network = "udp"
	}
	conn, err := net.ListenUDP(network, rawConn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		return nil, fmt.Errorf("could not listen UDP: %w", err)
	}
	return kcp.NewConn3(1, raddr, block, 10, 3, conn)
}
