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
	PingProvider(ip string, providerPort, consumerPort, proxyPort int, stop <-chan struct{}) error
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
	natProxy       *NATProxy
	eventPublisher eventbus.Publisher
}

// PortSupplier provides port needed to run a service on
type PortSupplier interface {
	Acquire() (port.Port, error)
}

// NewPinger returns Pinger instance
func NewPinger(pingConfig *PingConfig, proxy *NATProxy, publisher eventbus.Publisher) NATPinger {
	return &Pinger{
		pingConfig:     pingConfig,
		pingTarget:     make(chan *Params),
		stop:           make(chan struct{}),
		stopNATProxy:   make(chan struct{}),
		natProxy:       proxy,
		eventPublisher: publisher,
	}
}

// Params contains session parameters needed to NAT ping remote peer
type Params struct {
	ProviderPort        int
	ConsumerPort        int
	ConsumerPublicIP    string
	ProxyPortMappingKey string
	Cancel              chan struct{}
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
	return params.ConsumerPort > 0
}

// Stop stops pinger loop
func (p *Pinger) Stop() {
	p.once.Do(func() {
		close(p.stopNATProxy)
		close(p.stop)
	})
}

// PingProvider pings provider determined by destination provided in sessionConfig
func (p *Pinger) PingProvider(ip string, providerPort, consumerPort, proxyPort int, stop <-chan struct{}) error {
	log.Info().Msg("NAT pinging to provider")

	conn, err := p.getConnection(ip, providerPort, consumerPort)
	if err != nil {
		return errors.Wrap(err, "failed to get connection")
	}

	// Add read deadline to prevent possible conn.Read hang when remote peer doesn't send ping ack.
	conn.SetReadDeadline(time.Now().Add(p.pingConfig.Timeout * 2))

	pingStop := make(chan struct{})
	defer close(pingStop)
	go func() {
		err := p.ping(conn, pingStop)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	time.Sleep(p.pingConfig.Interval)
	err = p.pingReceiver(conn, stop)
	if err != nil {
		return err
	}

	// send one last ping request to end hole punching procedure gracefully
	err = p.sendPingRequest(conn, 128)
	if err != nil {
		return errors.Wrap(err, "remote ping failed")
	}

	if proxyPort > 0 {
		consumerAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
		log.Info().Msg("Handing connection to consumer NATProxy: " + consumerAddr)

		// Set higher read deadline when NAT proxy is used.
		conn.SetReadDeadline(time.Now().Add(12 * time.Hour))
		p.stopNATProxy = p.natProxy.consumerHandOff(consumerAddr, conn)
	} else {
		log.Info().Msg("Closing ping connection")
		if err := conn.Close(); err != nil {
			return errors.Wrap(err, "could not close ping conn")
		}
	}
	return nil
}

func (p *Pinger) ping(conn *net.UDPConn, stop <-chan struct{}) error {
	// Windows detects that 1 TTL is too low and throws an exception during send
	ttl := 0
	i := 0

	for {
		select {
		case <-stop:
			return nil

		case <-time.After(p.pingConfig.Interval):
			log.Debug().Msg("Pinging... ")
			// This is the essence of the TTL based udp punching.
			// We're slowly increasing the TTL so that the packet is held.
			// After a few attempts we're setting the value to 128 and assuming we're through.
			// We could stop sending ping to Consumer beyond 4 hops to prevent from possible Consumer's router's
			//  DOS block, but we plan, that Consumer at the same time will be Provider too in near future.
			ttl++

			if ttl > 4 {
				ttl = 128
			}

			err := p.sendPingRequest(conn, ttl)
			if err != nil {
				p.eventPublisher.Publish(event.Topic, event.BuildFailureEvent(StageName, err))
				return err
			}

			i++

			if time.Duration(i)*p.pingConfig.Interval > p.pingConfig.Timeout {
				err := errors.New("timeout while waiting for ping ack, trying to continue")
				p.eventPublisher.Publish(event.Topic, event.BuildFailureEvent(StageName, err))
				return err
			}
		}
	}
}

func (p *Pinger) sendPingRequest(conn *net.UDPConn, ttl int) error {
	err := ipv4.NewConn(conn).SetTTL(ttl)
	if err != nil {
		return errors.Wrap(err, "pinger setting ttl failed")
	}

	_, err = conn.Write([]byte("continuously pinging to " + conn.RemoteAddr().String()))
	return errors.Wrap(err, "pinging request failed")
}

func (p *Pinger) getConnection(ip string, port int, pingerPort int) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Remote socket: " + udpAddr.String())

	conn, err := net.DialUDP("udp", &net.UDPAddr{Port: pingerPort}, udpAddr)
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
			n, err := conn.Read(buf)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to read remote peer: %s - attempting to continue", conn.RemoteAddr().String())
				continue
			}

			if n > 0 {
				log.Info().Msgf("Remote peer data received: %s, len: %d", string(buf[:n]), n)
				return nil
			}
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

	log.Info().Msgf("Ping target received: IP: %v, port: %v", pingParams.ConsumerPublicIP, pingParams.ConsumerPort)
	if !p.natProxy.isAvailable(pingParams.ProxyPortMappingKey) {
		log.Warn().Msgf("NATProxy is not available for this transport protocol key %v", pingParams.ProxyPortMappingKey)
		return
	}

	conn, err := p.getConnection(pingParams.ConsumerPublicIP, pingParams.ConsumerPort, pingParams.ProviderPort)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get connection")
		return
	}

	pingStop := make(chan struct{})
	defer close(pingStop)
	go func() {
		err := p.ping(conn, pingStop)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	err = p.pingReceiver(conn, pingParams.Cancel)
	if err != nil {
		log.Error().Err(err).Msg("Ping receiver error")
		return
	}

	p.eventPublisher.Publish(event.Topic, event.BuildSuccessfulEvent(StageName))

	log.Info().Msg("Ping received, waiting for a new connection")

	go p.natProxy.handOff(pingParams.ProxyPortMappingKey, conn)
}
