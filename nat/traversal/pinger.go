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
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/ipv4"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/event"
)

// StageName represents hole-punching stage of NAT traversal
const StageName = "hole_punching"

const (
	bufferLen = 2048 * 1024

	maxTTL   = 128
	msgOK    = "OK"
	msgOKACK = "OK_ACK"
)

var errNATPunchAttemptStopped = errors.New("NAT punch attempt stopped")

// NATPinger is responsible for pinging nat holes
type NATPinger interface {
	PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	PingConsumerPeer(ctx context.Context, id string, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
	Stop()
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
	stop           chan struct{}
	once           sync.Once
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
		eventPublisher: publisher,
	}
}

// Stop stops pinger loop
func (p *Pinger) Stop() {
	p.once.Do(func() {
		close(p.stop)
	})
}

// PingConsumerPeer pings remote peer with a defined configuration
// and notifies peer which connections will be used.
// It returns n connections if possible or error.
func (p *Pinger) PingConsumerPeer(ctx context.Context, id string, ip string, localPorts, remotePorts []int, initialTTL int, n int) ([]*net.UDPConn, error) {
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
			p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildFailureEvent(id, StageName, ctx.Err()))
			return nil, fmt.Errorf("ping failed: %w", ctx.Err())
		case ping := <-pingsCh:
			pings = append(pings, ping)
			if len(pings) == n {
				p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(id, StageName))
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
	ok := make(chan struct{}, 1)
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
		conn.Close()
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

// Valid returns that this pinger is a valid pinger
func (p *Pinger) Valid() bool {
	return true
}

type pingResponse struct {
	conn *net.UDPConn
	err  error
	id   int
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
