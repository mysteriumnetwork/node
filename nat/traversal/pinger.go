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

var (
	errNATPunchAttemptStopped  = errors.New("NAT punch attempt stopped")
	errNATPunchAttemptTimedOut = errors.New("NAT punch attempt timed out")
)

// NATProviderPinger pings provider and optionally hands off connection to consumer proxy.
type NATProviderPinger interface {
	PingProvider(ip string, cPorts, pPorts []int, proxyPort int) (*net.UDPConn, error)
}

// NATPinger is responsible for pinging nat holes
type NATPinger interface {
	NATProviderPinger
	PingTarget(*Params)
	BindServicePort(key string, port int)
	Start()
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
	pingTarget     chan *Params
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
		pingTarget:     make(chan *Params),
		stop:           make(chan struct{}),
		stopNATProxy:   make(chan struct{}),
		natProxy:       newNATProxy(),
		eventPublisher: publisher,
	}
}

// Params contains session parameters needed to NAT ping remote peer
type Params struct {
	ProviderPorts       []int
	ConsumerPorts       []int
	IP                  string
	ProxyPortMappingKey string
}

// Start starts NAT pinger and waits for PingTarget to ping
func (p *Pinger) Start() {
	log.Info().Msg("Starting a NAT pinger")

	for {
		select {
		case <-p.stop:
			log.Info().Msg("NAT pinger is stopped")
			return
		case pingParams := <-p.pingTarget:
			if isPunchingRequired(pingParams) {
				go p.pingTargetConsumer(pingParams)
			}
		}
	}
}

func isPunchingRequired(params *Params) bool {
	return true
}

// Stop stops pinger loop
func (p *Pinger) Stop() {
	p.once.Do(func() {
		close(p.stopNATProxy)
		close(p.stop)
	})
}

// PingProvider pings provider determined by destination provided in sessionConfig
func (p *Pinger) PingProvider(ip string, cPorts, pPorts []int, proxyPort int) (*net.UDPConn, error) {
	log.Info().Msg("NAT pinging to provider")

	stop := make(chan struct{})
	defer close(stop)

	conn, err := p.multiPing(ip, cPorts, pPorts, 128, stop)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return nil, err
	}

	consumerAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	log.Info().Msg("Handing connection to consumer NATProxy: " + consumerAddr)

	p.stopNATProxy = p.natProxy.consumerHandOff(consumerAddr, conn)

	return conn, err
}

func (p *Pinger) ping(conn *net.UDPConn, ttl int, stop <-chan struct{}) error {
	// Windows detects that 1 TTL is too low and throws an exception during send
	i := 0

	err := ipv4.NewConn(conn).SetTTL(ttl)
	if err != nil {
		return errors.Wrap(err, "pinger setting ttl failed")
	}

	for {
		select {
		case <-stop:
			return nil

		case <-time.After(p.pingConfig.Interval):
			log.Debug().Msgf("Pinging %s from %s...", conn.RemoteAddr().String(), conn.LocalAddr().String())

			_, err := conn.Write([]byte("continuously pinging to " + conn.RemoteAddr().String()))
			if err != nil {
				p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildFailureEvent(StageName, err))
				return errors.Wrap(err, "pinging request failed")
			}

			i++

			if time.Duration(i)*p.pingConfig.Interval > p.pingConfig.Timeout {
				err := errors.New("timeout while waiting for ping ack, trying to continue")
				p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildFailureEvent(StageName, err))
				return err
			}
		}
	}
}

func (p *Pinger) getConnection(ip string, remotePort int, localPort int) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ip, remotePort))
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Remote socket: " + udpAddr.String())

	conn, err := net.DialUDP("udp", &net.UDPAddr{Port: localPort}, udpAddr)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Local socket: " + conn.LocalAddr().String())

	return conn, nil
}

// PingTarget relays ping target address data
func (p *Pinger) PingTarget(target *Params) {
	select {
	case p.pingTarget <- target:
		return
	// do not block if ping target is not received
	case <-time.After(100 * time.Millisecond):
		log.Info().Msgf("Ping target timeout: %v", target)
		return
	}
}

// BindServicePort register service port to forward connection to
func (p *Pinger) BindServicePort(key string, port int) {
	p.natProxy.registerServicePort(key, port)
}

func (p *Pinger) pingReceiver(conn *net.UDPConn, stop <-chan struct{}) error {
	timeout := time.After(p.pingConfig.Timeout)
	buf := make([]byte, bufferLen)

	for {
		select {
		case <-timeout:
			return errNATPunchAttemptTimedOut
		case <-stop:
			return errNATPunchAttemptStopped
		default:
			// Add read deadline to prevent possible conn.Read hang when remote peer doesn't send ping ack.
			conn.SetReadDeadline(time.Now().Add(p.pingConfig.Timeout * 2))
			n, err := conn.Read(buf)
			// Set higher read deadline when NAT proxy is used.
			conn.SetReadDeadline(time.Now().Add(12 * time.Hour))
			if err != nil || n == 0 {
				log.Error().Err(err).Msgf("Failed to read remote peer: %s - attempting to continue", conn.RemoteAddr().String())
				continue
			}

			log.Info().Msgf("Remote peer data received: %s, len: %d", string(buf[:n]), n)
			return nil
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

func (p *Pinger) pingTargetConsumer(pingParams *Params) {
	log.Info().Msgf("Pinging peer with: %+v", pingParams)

	if pingParams.ProxyPortMappingKey == "" {
		log.Error().Msg("Service proxy connection port mapping key is missing")
		return
	}

	stop := make(chan struct{})
	defer close(stop)

	conn, err := p.multiPing(pingParams.IP, pingParams.ProviderPorts, pingParams.ConsumerPorts, 2, stop)
	if err != nil {
		log.Err(err).Msg("Failed to ping remote peer")
		return
	}

	err = ipv4.NewConn(conn).SetTTL(128)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set connection TTL")
		return
	}

	conn.Write([]byte("Using this connection"))

	p.eventPublisher.Publish(event.AppTopicTraversal, event.BuildSuccessfulEvent(StageName))
	log.Info().Msg("Ping received, waiting for a new connection")

	go p.natProxy.handOff(pingParams.ProxyPortMappingKey, conn)
}

func (p *Pinger) multiPing(remoteIP string, localPorts, remotePorts []int, initialTTL int, stop <-chan struct{}) (*net.UDPConn, error) {
	if len(localPorts) != len(remotePorts) {
		return nil, errors.New("number of local and remote ports does not match")
	}

	type res struct {
		conn *net.UDPConn
		err  error
	}

	ch := make(chan res, len(localPorts))

	for i := range localPorts {
		go func(i int) {
			conn, err := p.singlePing(remoteIP, localPorts[i], remotePorts[i], initialTTL+i, stop)
			ch <- res{conn, err}
		}(i)
	}

	// First responce wins. Other are not important.
	r := <-ch
	return r.conn, r.err
}

func (p *Pinger) singlePing(remoteIP string, localPort, remotePort, ttl int, stop <-chan struct{}) (*net.UDPConn, error) {
	conn, err := p.getConnection(remoteIP, remotePort, localPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get connection")
	}

	go func() {
		err := p.ping(conn, ttl, stop)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	err = p.pingReceiver(conn, stop)
	return conn, errors.Wrap(err, "ping receiver error")
}
