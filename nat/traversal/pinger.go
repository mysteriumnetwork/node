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

const maxTTL = 128

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
	PingPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
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
	conns, err := p.PingPeer(ip, localPorts, remotePorts, maxTTL, 1)
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
	conns, err := p.PingPeer(ip, localPorts, remotePorts, 2, 1)
	if err != nil {
		log.Error().Err(err).Msg("Failed to ping remote peer")
		return
	}

	conn := conns[0]

	p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(StageName))
	log.Info().Msgf("Ping received from: %s, sending OK from: %s", conn.RemoteAddr(), conn.LocalAddr())

	go p.natProxy.handOff(mappingKey, conn)
}

// PingPeer pings remote peer with a defined configuration.
// It returns n connections if possible or all available with error.
func (p *Pinger) PingPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
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

func (p *Pinger) ping(conn *net.UDPConn, remoteAddr *net.UDPAddr, ttl int, stop <-chan struct{}) error {
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

		case <-time.After(p.pingConfig.Interval):
			log.Debug().Msgf("Pinging %s from %s...", remoteAddr, conn.LocalAddr())

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

			ch <- pingResponse{conn, err}

			wg.Done()
		}(i)
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

	go func() {
		err := p.ping(conn, remoteAddr, ttl, stop)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	laddr := conn.LocalAddr().(*net.UDPAddr)
	raddr, err := p.pingReceiver(conn, stop)
	if err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "ping receiver error")
	}

	conn.Close()

	return net.DialUDP("udp4", laddr, raddr)
}
