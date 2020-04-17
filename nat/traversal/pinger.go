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
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"errors"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/ipv4"
)

// StageName represents hole-punching stage of NAT traversal
const StageName = "hole_punching"

const (
	maxTTL   = 128
	msgOK    = "OK"
	msgOKACK = "OK_ACK"
)

var (
	errNATPunchAttemptStopped = errors.New("NAT punch attempt stopped")
)

// NATProviderPinger pings provider and optionally hands off connection to consumer proxy.
type NATProviderPinger interface {
	PingProvider(ctx context.Context, ip string, localPorts, remotePorts []int, proxyPort int) (localPort, remotePort int, err error)
}

// NATPinger is responsible for pinging nat holes
type NATPinger interface {
	NATProviderPinger
	PingConsumer(ctx context.Context, ip string, localPorts, remotePorts []int, mappingKey string)
	PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	PingConsumerPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	BindServicePort(key string, port int)
	Stop()
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
	Valid() bool
}

// PingConfig represents NAT pinger config.
type PingConfig struct {
	Interval            time.Duration
	Timeout             time.Duration
	SendConnACKInterval time.Duration
}

// DefaultPingConfig returns default NAT pinger config.
func DefaultPingConfig() *PingConfig {
	return &PingConfig{
		Interval:            50 * time.Millisecond,
		Timeout:             10 * time.Second,
		SendConnACKInterval: 200 * time.Millisecond,
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
		natProxy:       NewNATProxy(),
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
func (p *Pinger) PingProvider(ctx context.Context, ip string, localPorts, remotePorts []int, proxyPort int) (localPort, remotePort int, err error) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	conns, err := p.pingPeer(ctx, ip, localPorts, remotePorts, maxTTL, 1)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to ping remote peer: %w", err)
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

		p.stopNATProxy = p.natProxy.ConsumerHandOff(consumerAddr, conn)
	} else {
		conn.Close()
	}

	return localPort, remotePort, nil
}

// PingConsumer pings consumer with increasing TTL for every connection.
func (p *Pinger) PingConsumer(ctx context.Context, ip string, localPorts, remotePorts []int, mappingKey string) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	conns, err := p.pingPeer(ctx, ip, localPorts, remotePorts, 2, 1)
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
func (p *Pinger) pingPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	log.Info().Msg("NAT pinging to remote peer")

	ch, err := p.multiPing(ctx, ip, localPorts, remotePorts, initialTTL)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res, more := <-ch:
			if !more {
				return nil, errors.New("not enough connections")
			}

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
	}
}

// PingConsumerPeer pings remote peer with a defined configuration
// and notifies peer which connections will be used.
// It returns n connections if possible or error.
func (p *Pinger) PingConsumerPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	log.Info().Msg("NAT pinging to remote peer")

	stop := make(chan struct{})
	defer close(stop)

	ch, err := p.multiPingN(ctx, ip, localPorts, remotePorts, initialTTL, n)
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

			if err := p.sendConnACK(ctx, res.conn); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Warn().Err(err).Msg("Failed to send conn ACK to consumer")
				}
			} else {
				pingsCh <- res
			}
		}
	}()

	var pings []pingResponse
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ping failed: %w", ctx.Err())
		case ping := <-pingsCh:
			pings = append(pings, ping)
			if len(pings) == n {
				return sortedConns(pings), nil
			}
		}
	}
}

// PingProviderPeer pings remote peer with a defined configuration
// and waits for peer to send ack with connection selected ids.
// It returns n connections if possible or error.
func (p *Pinger) PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	log.Info().Msg("NAT pinging to remote peer")

	ch, err := p.multiPingN(ctx, ip, localPorts, remotePorts, initialTTL, n)
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
				if err := waitMsg(ctx, ping.conn, msgOK); err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Err(err).Msg("Failed to wait for conn OK from provider")
					}
					return
				}
				sendMsg(ping.conn, msgOKACK)
				pingsCh <- ping
			}(res)
		}
	}()

	var pings []pingResponse
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ping failed: %w", ctx.Err())
		case ping := <-pingsCh:
			pings = append(pings, ping)
			if len(pings) == n {
				return sortedConns(pings), nil
			}
		}
	}
}

// sendConnACK notifies peer that we are using this connection
// and waits for ack or returns timeout err.
func (p *Pinger) sendConnACK(ctx context.Context, conn *net.UDPConn) error {
	ackWaitErr := make(chan error)
	go func() {
		ackWaitErr <- waitMsg(ctx, conn, msgOKACK)
	}()

	for {
		select {
		case err := <-ackWaitErr:
			return err
		case <-time.After(p.pingConfig.SendConnACKInterval):
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
func waitMsg(ctx context.Context, conn *net.UDPConn, msg string) error {
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
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for %q msg: %w", msg, ctx.Err())
	}
}

func sendMsg(conn *net.UDPConn, msg string) {
	conn.Write([]byte(msg))
}

func (p *Pinger) ping(ctx context.Context, conn *net.UDPConn, remoteAddr *net.UDPAddr, ttl int, pingReceived <-chan struct{}) error {
	err := ipv4.NewConn(conn).SetTTL(ttl)
	if err != nil {
		return fmt.Errorf("pinger setting ttl failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-p.stop:
			return nil
		case <-pingReceived:
			return nil

		case <-time.After(p.pingConfig.Interval):
			log.Trace().Msgf("Pinging %s from %s... with ttl %d", remoteAddr, conn.LocalAddr(), ttl)

			_, err := conn.WriteToUDP([]byte("continuously pinging to "+remoteAddr.String()), remoteAddr)
			if err != nil {
				return fmt.Errorf("pinging request failed: %w", err)
			}
		}
	}
}

// BindServicePort register service port to forward connection to
func (p *Pinger) BindServicePort(key string, port int) {
	p.natProxy.registerServicePort(key, port)
}

func (p *Pinger) pingReceiver(ctx context.Context, conn *net.UDPConn) (*net.UDPAddr, error) {
	buf := make([]byte, bufferLen)

	for {
		select {
		case <-ctx.Done():
			return nil, errNATPunchAttemptStopped
		case <-p.stop:
			return nil, errNATPunchAttemptStopped
		default:
			// Add read deadline to prevent possible conn.Read hang when remote peer doesn't send ping ack.
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
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
	p.natProxy.SetProtectSocketCallback(socketProtect)
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

func (p *Pinger) multiPing(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int) (<-chan pingResponse, error) {
	if len(localPorts) != len(remotePorts) {
		return nil, errors.New("number of local and remote ports does not match")
	}

	var wg sync.WaitGroup
	ch := make(chan pingResponse, len(localPorts))

	for i := range localPorts {
		wg.Add(1)

		go func(i int) {
			conn, err := p.singlePing(ctx, ip, localPorts[i], remotePorts[i], initialTTL+i)

			ch <- pingResponse{id: i, conn: conn, err: err}

			wg.Done()
		}(i)
	}

	go func() { wg.Wait(); close(ch) }()

	return ch, nil
}

func (p *Pinger) multiPingN(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (<-chan pingResponse, error) {
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
			conn, err := p.singlePing(ctx, ip, localPorts[i], remotePorts[i], ttl)
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

func (p *Pinger) singlePing(ctx context.Context, remoteIP string, localPort, remotePort, ttl int) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: localPort})
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	log.Info().Msgf("Local socket: %s", conn.LocalAddr())

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteIP, remotePort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve remote addres: %w", err)
	}

	pingReceived := make(chan struct{}, 1)
	go func() {
		err := p.ping(ctx, conn, remoteAddr, ttl, pingReceived)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	laddr := conn.LocalAddr().(*net.UDPAddr)
	raddr, err := p.pingReceiver(ctx, conn)
	pingReceived <- struct{}{}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping receiver error: %w", err)
	}

	conn.Close()

	return net.DialUDP("udp4", laddr, raddr)
}
