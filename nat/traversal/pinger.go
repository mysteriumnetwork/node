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
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/ipv4"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/router"
)

// StageName represents hole-punching stage of NAT traversal
const StageName = "hole_punching"

const (
	bufferLen = 64

	maxTTL            = 128
	msgOK             = "OK"
	msgOKACK          = "OK_ACK"
	msgPing           = "continuously pinging to "
	sendRetryInterval = 5 * time.Millisecond
	sendRetries       = 10
)

// ErrTooFew indicates there were too few successful ping
// responses to build requested number of connections
var ErrTooFew = errors.New("too few connections were built")

// NATPinger is responsible for pinging nat holes
type NATPinger interface {
	PingProviderPeer(ctx context.Context, localIP, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	PingConsumerPeer(ctx context.Context, id string, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
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
		Interval:            5 * time.Millisecond,
		Timeout:             10 * time.Second,
		SendConnACKInterval: 100 * time.Millisecond,
	}
}

// Pinger represents NAT pinger structure
type Pinger struct {
	pingConfig     *PingConfig
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
		eventPublisher: publisher,
	}
}

func drainPingResponses(responses <-chan pingResponse) {
	for response := range responses {
		log.Warn().Err(response.err).Msgf("Sanitizing ping response on %#v", response)
		if response.conn != nil {
			response.conn.Close()
			log.Warn().Msgf("Collected dangling socket: %s", response.conn.LocalAddr().String())
		}
	}
}

func cleanupConnections(responses []pingResponse) {
	for _, response := range responses {
		response.conn.Close()
	}
}

// PingConsumerPeer pings remote peer with a defined configuration
// and notifies peer which connections will be used.
// It returns n connections if possible or error.
func (p *Pinger) PingConsumerPeer(ctx context.Context, id string, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	log.Info().Msg("NAT pinging to remote peer")

	stop := make(chan struct{})
	defer close(stop)

	ch, err := p.multiPingN(ctx, "", remoteIP, localPorts, remotePorts, initialTTL, n)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	pingsCh := make(chan pingResponse, n)
	go func() {
		var wg sync.WaitGroup
		for res := range ch {
			if res.err != nil {
				if !errors.Is(res.err, context.Canceled) {
					log.Warn().Err(res.err).Msg("One of the pings has error")
				}
				continue
			}

			if err := ipv4.NewConn(res.conn).SetTTL(maxTTL); err != nil {
				res.conn.Close()
				log.Warn().Err(res.err).Msg("Failed to set connection TTL")
				continue
			}

			p.sendMsg(res.conn, msgOK) // Notify peer that we are using this connection.

			wg.Add(1)
			go func(ping pingResponse) {
				defer wg.Done()
				if err := p.sendConnACK(ctx, ping.conn); err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Warn().Err(err).Msg("Failed to send conn ACK to consumer")
					}
					ping.conn.Close()
				} else {
					pingsCh <- ping
				}
			}(res)
		}
		wg.Wait()
		close(pingsCh)
	}()

	var pings []pingResponse
	for ping := range pingsCh {
		pings = append(pings, ping)
		if len(pings) == n {
			p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(id, StageName))
			go drainPingResponses(pingsCh)
			return sortedConns(pings), nil
		}
	}

	p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildFailureEvent(id, StageName, ErrTooFew))
	cleanupConnections(pings)
	return nil, ErrTooFew
}

// PingProviderPeer pings remote peer with a defined configuration
// and waits for peer to send ack with connection selected ids.
// It returns n connections if possible or error.
func (p *Pinger) PingProviderPeer(ctx context.Context, localIP, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.pingConfig.Timeout)
	defer cancel()

	log.Info().Msg("NAT pinging to remote peer")

	ch, err := p.multiPingN(ctx, localIP, remoteIP, localPorts, remotePorts, initialTTL, n)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	pingsCh := make(chan pingResponse, len(localPorts))
	go func() {
		var wg sync.WaitGroup
		for res := range ch {
			if res.err != nil {
				if !errors.Is(res.err, context.Canceled) {
					log.Warn().Err(res.err).Msg("One of the pings has error")
				}
				continue
			}

			if err := ipv4.NewConn(res.conn).SetTTL(maxTTL); err != nil {
				res.conn.Close()
				log.Warn().Err(res.err).Msg("Failed to set connection TTL")
				continue
			}

			p.sendMsg(res.conn, msgOK) // Notify peer that we are using this connection.

			// Wait for peer to notify that it uses this connection too.
			wg.Add(1)
			go func(ping pingResponse) {
				defer wg.Done()
				if err := p.waitMsg(ctx, ping.conn, msgOK); err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Err(err).Msg("Failed to wait for conn OK from provider")
					}
					ping.conn.Close()
					return
				}
				pingsCh <- ping
			}(res)
		}
		wg.Wait()
		close(pingsCh)
	}()

	var pings []pingResponse
	for ping := range pingsCh {
		pings = append(pings, ping)
		p.sendMsg(ping.conn, msgOKACK)
		if len(pings) == n {
			go drainPingResponses(pingsCh)
			return sortedConns(pings), nil
		}
	}

	cleanupConnections(pings)
	return nil, ErrTooFew
}

// sendConnACK notifies peer that we are using this connection
// and waits for ack or returns timeout err.
func (p *Pinger) sendConnACK(ctx context.Context, conn *net.UDPConn) error {
	ackWaitErr := make(chan error)
	go func() {
		ackWaitErr <- p.waitMsg(ctx, conn, msgOKACK)
	}()

	for {
		select {
		case err := <-ackWaitErr:
			return err
		case <-time.After(p.pingConfig.SendConnACKInterval):
			p.sendMsg(conn, msgOK)
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
func (p *Pinger) waitMsg(ctx context.Context, conn *net.UDPConn, msg string) error {
	var (
		n   int
		err error
	)
	buf := make([]byte, 1024)
	// just reasonable upper boundary for receive errors to not enter infinite
	// loop on closed socket, but still skim errors of closed port etc
	// +1 in denominator is to avoid division by zero
	recvErrLimit := 2 * int(p.pingConfig.Timeout/(p.pingConfig.Interval+1))
	for errCount := 0; errCount < recvErrLimit; {
		n, err = readFromConnWithContext(ctx, conn, buf)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// process returned data unconditionally as io.Reader dictates to
		v := string(buf[:n])
		if v == msg {
			return nil
		}
		if err != nil {
			errCount++
			log.Error().Err(err).Msgf("got error in waitMsg, trying to recover. %d attempts left",
				recvErrLimit-errCount)
			continue
		}
	}
	return fmt.Errorf("too many recv errors, last one: %w", err)
}

func (p *Pinger) sendMsg(conn *net.UDPConn, msg string) {
	for i := 0; i < sendRetries; i++ {
		_, err := conn.Write([]byte(msg))
		if err != nil {
			log.Error().Err(err).Msg("pinger message send failed")
			time.Sleep(p.pingConfig.Interval)
		} else {
			return
		}
	}
}

func (p *Pinger) ping(ctx context.Context, conn *net.UDPConn, remoteAddr *net.UDPAddr, ttl int) error {
	err := ipv4.NewConn(conn).SetTTL(ttl)
	if err != nil {
		return fmt.Errorf("pinger setting ttl failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(p.pingConfig.Interval):
			_, err := conn.WriteToUDP([]byte(msgPing+remoteAddr.String()), remoteAddr)
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				return fmt.Errorf("pinging request failed: %w", err)
			}
		}
	}
}

func readFromConnWithContext(ctx context.Context, conn net.Conn, buf []byte) (n int, err error) {
	readDone := make(chan struct{})
	go func() {
		n, err = conn.Read(buf)
		close(readDone)
	}()

	select {
	case <-ctx.Done():
		conn.SetReadDeadline(time.Unix(0, 0))
		<-readDone
		conn.SetReadDeadline(time.Time{})
		return 0, ctx.Err()
	case <-readDone:
		return
	}
}

func readFromUDPWithContext(ctx context.Context, conn *net.UDPConn, buf []byte) (n int, from *net.UDPAddr, err error) {
	readDone := make(chan struct{})
	go func() {
		n, from, err = conn.ReadFromUDP(buf)
		close(readDone)
	}()

	select {
	case <-ctx.Done():
		conn.SetReadDeadline(time.Unix(0, 0))
		<-readDone
		conn.SetReadDeadline(time.Time{})
		return 0, nil, ctx.Err()
	case <-readDone:
		return
	}
}

func (p *Pinger) pingReceiver(ctx context.Context, conn *net.UDPConn) (*net.UDPAddr, error) {
	buf := make([]byte, bufferLen)

	for {
		n, raddr, err := readFromUDPWithContext(ctx, conn, buf)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if err != nil || n == 0 {
			log.Debug().Err(err).Msgf("Failed to read remote peer: %s - attempting to continue", raddr)
			continue
		}

		msg := string(buf[:n])
		log.Debug().Msgf("Remote peer data received, len: %d", n)

		if msg == msgOK || strings.HasPrefix(msg, msgPing) {
			return raddr, nil
		}

		log.Debug().Err(err).Msgf("Unexpected message: %s - attempting to continue", msg)
	}
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

func (p *Pinger) multiPingN(ctx context.Context, localIP, remoteIP string, localPorts, remotePorts []int, initialTTL int, n int) (<-chan pingResponse, error) {
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
			conn, err := p.singlePing(ctx, localIP, remoteIP, localPorts[i], remotePorts[i], ttl)
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

func (p *Pinger) singlePing(ctx context.Context, localIP, remoteIP string, localPort, remotePort, ttl int) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(localIP), Port: localPort})
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	defer conn.Close()

	if err := router.ProtectUDPConn(conn); err != nil {
		return nil, fmt.Errorf("failed to protect udp connection: %w", err)
	}

	log.Debug().Msgf("Local socket: %s", conn.LocalAddr())

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteIP, remotePort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve remote address: %w", err)
	}

	ctx1, cl := context.WithCancel(ctx)
	go func() {
		err := p.ping(ctx1, conn, remoteAddr, ttl)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	laddr := conn.LocalAddr().(*net.UDPAddr)
	raddr, err := p.pingReceiver(ctx, conn)
	cl()
	if err != nil {
		return nil, fmt.Errorf("ping receiver error: %w", err)
	}
	// need to dial same connection further
	conn.Close()

	newConn, err := net.DialUDP("udp4", laddr, raddr)
	if err != nil {
		return nil, err
	}

	if err := router.ProtectUDPConn(newConn); err != nil {
		newConn.Close()
		return nil, fmt.Errorf("failed to protect udp connection: %w", err)
	}

	return newConn, nil
}
