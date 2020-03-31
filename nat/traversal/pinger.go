/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package traversal

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/ipv4"
)

// StageName represents hole-punching stage of NAT traversal
const StageName = "hole_punching"

const (
	maxTTL              = 128
	msgOK               = "OK"
	msgOKACK            = "OK_ACK"
	msgReceiveTimeout   = 2 * time.Second
	sendConnACKInterval = 100 * time.Millisecond
)

var (
	errNATPunchAttemptStopped  = errors.New("NAT punch attempt stopped")
	errNATPunchAttemptTimedOut = errors.New("NAT punch attempt timed out")
)

// NATProviderPinger pings provider and optionally hands off connection to consumer proxy.
type NATProviderPinger interface {
	PingProvider(ip string, localPorts, remotePorts []int, proxyPort int) (localPort, remotePort int, err error)
}

// NATPinger is responsible for pinging nat holes
type NATPinger interface {
	NATProviderPinger
	PingConsumer(ip string, localPorts, remotePorts []int, mappingKey string)
	PingProviderPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	PingConsumerPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	BindServicePort(key string, port int)
	Stop()
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
	Valid() bool
}

// PingConfig represents NAT pinger config.
type PingConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// DefaultPingConfig returns default NAT pinger config.
func DefaultPingConfig() *PingConfig {
	return &PingConfig{
		Interval: 200 * time.Millisecond,
		Timeout:  10 * time.Second,
	}
}

// Pinger represents NAT pinger structure
type Pinger struct {
	pingConfig     *PingConfig
	stop           chan struct{}
	stopNATProxy   chan struct{}
	once           sync.Once
	natProxy       *natProxy
	eventPublisher eventbus.Publisher
}

// PortSupplier provides port needed to run a service on
type PortSupplier interface {
	Acquire() (port.Port, error)
}

// NewPinger returns Pinger instance
func NewPinger(pingConfig *PingConfig, publisher eventbus.Publisher) NATPinger {
	return &Pinger{
		pingConfig:     pingConfig,
		stop:           make(chan struct{}),
		stopNATProxy:   make(chan struct{}),
		natProxy:       newNATProxy(),
		eventPublisher: publisher,
	}
}

// Params contains session parameters needed to NAT ping remote peer
type Params struct {
	RemotePorts         []int
	LocalPorts          []int
	IP                  string
	ProxyPortMappingKey string
}

// Stop stops pinger loop
func (p *Pinger) Stop() {
	p.once.Do(func() {
		close(p.stopNATProxy)
		close(p.stop)
	})
}

// PingProvider pings provider determined by destination provided in sessionConfig
func (p *Pinger) PingProvider(ip string, localPorts, remotePorts []int, proxyPort int) (localPort, remotePort int, err error) {
	conns, err := p.pingPeer(ip, localPorts, remotePorts, maxTTL, 1)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to ping remote peer")
	}

	conn := conns[0]
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		localPort = addr.Port
	}

	if addr, ok := conn.RemoteAddr().(*net.UDPAddr); ok {
		remotePort = addr.Port
	}

	if proxyPort > 0 {
		consumerAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
		log.Info().Msg("Handing connection to consumer NATProxy: " + consumerAddr)

		p.stopNATProxy = p.natProxy.consumerHandOff(consumerAddr, conn)
	} else {
		conn.Close()
	}

	return localPort, remotePort, nil
}

// PingConsumer pings consumer with increasing TTL for every connection.
func (p *Pinger) PingConsumer(ip string, localPorts, remotePorts []int, mappingKey string) {
	conns, err := p.pingPeer(ip, localPorts, remotePorts, 2, 1)
	if err != nil {
		log.Error().Err(err).Msg("Failed to ping remote peer")
		return
	}

	conn := conns[0]

	p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(StageName))
	log.Info().Msgf("Ping received from: %s, sending OK from: %s", conn.RemoteAddr(), conn.LocalAddr())

	go p.natProxy.handOff(mappingKey, conn)
}

// pingPeer pings remote peer with a defined configuration.
// It returns n connections if possible or all available with error.
func (p *Pinger) pingPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	log.Info().Msg("NAT pinging to remote peer")

	stop := make(chan struct{})
	defer close(stop)

	ch, err := p.multiPing(ip, localPorts, remotePorts, initialTTL, stop)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	for res := range ch {
		if res.err != nil {
			log.Warn().Err(res.err).Msg("One of the pings has error")
		} else {
			if err := ipv4.NewConn(res.conn).SetTTL(maxTTL); err != nil {
				log.Warn().Err(res.err).Msg("Failed to set connection TTL")
				continue
			}

			res.conn.Write([]byte("OK")) // notify peer that we are using this connection.

			conns = append(conns, res.conn)
			if len(conns) == n {
				return conns, nil
			}
		}
	}

	return conns, errors.New("not enough connections")
}

// PingConsumerPeer pings remote peer with a defined configuration
// and notifies peer which connections will be used.
// It returns n connections if possible or error.
func (p *Pinger) PingConsumerPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	log.Info().Msg("NAT pinging to remote peer")

	stop := make(chan struct{})
	defer close(stop)

	ch, err := p.multiPingN(ip, localPorts, remotePorts, initialTTL, n, stop)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	var pings []pingResponse
	for res := range ch {
		if res.err != nil {
			log.Warn().Err(res.err).Msg("One of the pings has error")
			continue
		}

		if err := ipv4.NewConn(res.conn).SetTTL(maxTTL); err != nil {
			log.Warn().Err(res.err).Msg("Failed to set connection TTL")
			continue
		}

		sendMsg(res.conn, msgOK) // Notify peer that we are using this connection.

		pings = append(pings, res)
		if len(pings) == n {
			waitAllConnACKSent(pings)
			return sortedConns(pings), nil
		}
	}

	return nil, errors.New("not enough connections")
}

// PingProviderPeer pings remote peer with a defined configuration
// and waits for peer to send ack with connection selected ids.
// It returns n connections if possible or error.
func (p *Pinger) PingProviderPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	log.Info().Msg("NAT pinging to remote peer")

	stop := make(chan struct{})
	defer close(stop)

	ch, err := p.multiPingN(ip, localPorts, remotePorts, initialTTL, n, stop)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	pingsCh := make(chan pingResponse, n)
	go func() {
		for res := range ch {
			if res.err != nil {
				log.Warn().Err(res.err).Msg("One of the pings has error")
				continue
			}

			if err := ipv4.NewConn(res.conn).SetTTL(maxTTL); err != nil {
				log.Warn().Err(res.err).Msg("Failed to set connection TTL")
				continue
			}

			sendMsg(res.conn, msgOK) // Notify peer that we are using this connection.

			// Wait for peer to notify that it uses this connection too.
			go func(ping pingResponse) {
				if err := waitMsg(ping.conn, msgOK); err != nil {
					log.Err(err).Msg("Failed to wait for conn OK from provider")
					return
				}
				sendMsg(ping.conn, msgOKACK)
				pingsCh <- ping
			}(res)
		}
	}()

	timeout := time.After(p.pingConfig.Timeout)
	var pings []pingResponse
	for {
		select {
		case ping := <-pingsCh:
			pings = append(pings, ping)
			if len(pings) == n {
				return sortedConns(pings), nil
			}
		case <-timeout:
			return nil, errNATPunchAttemptTimedOut
		}
	}
}

func waitAllConnACKSent(pings []pingResponse) {
	var wg sync.WaitGroup
	wg.Add(len(pings))
	for _, ping := range pings {
		go func(conn *net.UDPConn) {
			defer wg.Done()
			if err := sendConnACK(conn); err != nil {
				log.Warn().Err(err).Msg("Failed to send conn ACK to consumer")
			}
		}(ping.conn)
	}
	wg.Wait()
}

// sendConnACK notifies peer that we are using this connection
// and waits for ack or returns timeout err.
func sendConnACK(conn *net.UDPConn) error {
	ackWaitErr := make(chan error)
	go func() {
		ackWaitErr <- waitMsg(conn, msgOKACK)
	}()

	for {
		select {
		case err := <-ackWaitErr:
			return err
		case <-time.After(sendConnACKInterval):
			sendMsg(conn, msgOK)
		}
	}
}

func sortedConns(pings []pingResponse) []*net.UDPConn {
	sort.Slice(pings, func(i, j int) bool {
		return pings[i].id < pings[j].id
	})
	var conns []*net.UDPConn
	for _, p := range pings {
		conns = append(conns, p.conn)
	}
	return conns
}

// waitMsg waits until conn receives given message or timeouts.
func waitMsg(conn *net.UDPConn, msg string) error {
	ok := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			v := string(buf[:n])
			if v == msg {
				ok <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-ok:
		return nil
	case <-time.After(msgReceiveTimeout):
		return fmt.Errorf("timeout waiting for %q msg", msg)
	}
}

func sendMsg(conn *net.UDPConn, msg string) {
	conn.Write([]byte(msg))
}

func (p *Pinger) ping(conn *net.UDPConn, remoteAddr *net.UDPAddr, ttl int, stop <-chan struct{}, pingReceived <-chan struct{}) error {
	err := ipv4.NewConn(conn).SetTTL(ttl)
	if err != nil {
		return errors.Wrap(err, "pinger setting ttl failed")
	}

	for deadline := time.Now().Add(p.pingConfig.Timeout); time.Now().Before(deadline); {
		select {
		case <-stop:
			return nil
		case <-p.stop:
			return nil
		case <-pingReceived:
			return nil

		case <-time.After(p.pingConfig.Interval):
			log.Trace().Msgf("Pinging %s from %s... with ttl %d", remoteAddr, conn.LocalAddr(), ttl)

			_, err := conn.WriteToUDP([]byte("continuously pinging to "+remoteAddr.String()), remoteAddr)
			if err != nil {
				return errors.Wrap(err, "pinging request failed")
			}
		}
	}

	return errors.New("timeout while waiting for ping ack, trying to continue")
}

// BindServicePort register service port to forward connection to
func (p *Pinger) BindServicePort(key string, port int) {
	p.natProxy.registerServicePort(key, port)
}

func (p *Pinger) pingReceiver(conn *net.UDPConn, stop <-chan struct{}) (*net.UDPAddr, error) {
	timeout := time.After(p.pingConfig.Timeout)
	buf := make([]byte, bufferLen)

	for {
		select {
		case <-timeout:
			return nil, errNATPunchAttemptTimedOut
		case <-stop:
			return nil, errNATPunchAttemptStopped
		case <-p.stop:
			return nil, errNATPunchAttemptStopped
		default:
			// Add read deadline to prevent possible conn.Read hang when remote peer doesn't send ping ack.
			conn.SetReadDeadline(time.Now().Add(p.pingConfig.Timeout * 2))
			n, raddr, err := conn.ReadFromUDP(buf)
			// Reset read deadline.
			conn.SetReadDeadline(time.Time{})

			if err != nil || n == 0 {
				log.Debug().Err(err).Msgf("Failed to read remote peer: %s - attempting to continue", raddr)
				continue
			}

			log.Info().Msgf("Remote peer data received: %s, len: %d, from: %s", string(buf[:n]), n, raddr)
			return raddr, nil
		}
	}
}

// SetProtectSocketCallback sets socket protection callback to be called when new socket is created in consumer NATProxy
func (p *Pinger) SetProtectSocketCallback(socketProtect func(socket int) bool) {
	p.natProxy.setProtectSocketCallback(socketProtect)
}

// Valid returns that this pinger is a valid pinger
func (p *Pinger) Valid() bool {
	return true
}

type pingResponse struct {
	conn *net.UDPConn
	err  error
	id   int
}

func (p *Pinger) multiPing(ip string, localPorts, remotePorts []int, initialTTL int, stop <-chan struct{}) (<-chan pingResponse, error) {
	if len(localPorts) != len(remotePorts) {
		return nil, errors.New("number of local and remote ports does not match")
	}

	var wg sync.WaitGroup
	ch := make(chan pingResponse, len(localPorts))

	for i := range localPorts {
		wg.Add(1)

		go func(i int) {
			conn, err := p.singlePing(ip, localPorts[i], remotePorts[i], initialTTL+i, stop)

			ch <- pingResponse{id: i, conn: conn, err: err}

			wg.Done()
		}(i)
	}

	go func() { wg.Wait(); close(ch) }()

	return ch, nil
}

func (p *Pinger) multiPingN(ip string, localPorts, remotePorts []int, initialTTL int, n int, stop <-chan struct{}) (<-chan pingResponse, error) {
	if len(localPorts) != len(remotePorts) {
		return nil, errors.New("number of local and remote ports does not match")
	}

	var wg sync.WaitGroup
	ch := make(chan pingResponse, len(localPorts))
	ttl := initialTTL
	resetTTL := initialTTL + (len(localPorts) / n)

	for i := range localPorts {
		wg.Add(1)

		go func(i, ttl int) {
			defer wg.Done()
			conn, err := p.singlePing(ip, localPorts[i], remotePorts[i], ttl, stop)
			ch <- pingResponse{conn: conn, err: err, id: i}
		}(i, ttl)

		// TTL increase is only needed for provider side which starts with low TTL value.
		if ttl < maxTTL {
			ttl++
		}
		if ttl == resetTTL {
			ttl = initialTTL
		}
	}

	go func() { wg.Wait(); close(ch) }()

	return ch, nil
}

func (p *Pinger) singlePing(remoteIP string, localPort, remotePort, ttl int, stop <-chan struct{}) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: localPort})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get connection")
	}

	log.Info().Msgf("Local socket: %s", conn.LocalAddr())

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteIP, remotePort))
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve remote address")
	}

	pingReceived := make(chan struct{}, 1)
	go func() {
		err := p.ping(conn, remoteAddr, ttl, stop, pingReceived)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	laddr := conn.LocalAddr().(*net.UDPAddr)
	raddr, err := p.pingReceiver(conn, stop)
	pingReceived <- struct{}{}
	if err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "ping receiver error")
	}

	conn.Close()

	return net.DialUDP("udp4", laddr, raddr)
}
