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
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/ipv4"
)

// StageName represents hole-punching stage of NAT traversal
const StageName = "hole_punching"
const pingInterval = 200
const pingTimeout = 10000

var (
	errNATPunchAttemptStopped  = errors.New("NAT punch attempt stopped")
	errNATPunchAttemptTimedOut = errors.New("NAT punch attempt timed out")
)

// NatPinger is responsible for pinging nat holes
type NatPinger interface {
	PingProvider(ip string, port int, consumerPort int, stop <-chan struct{}) error
	PingTarget(*Params)
	BindServicePort(key string, port int)
	Start()
	Stop()
	SetProtectSocketCallback(SocketProtect func(socket int) bool)
	StopNATProxy()
	Valid() bool
}

// Pinger represents NAT pinger structure
type Pinger struct {
	pingTarget     chan *Params
	pingCancelled  chan struct{}
	stop           chan struct{}
	stopNATProxy   chan struct{}
	once           sync.Once
	natEventWaiter NatEventWaiter
	natProxy       *NATProxy
	previousStage  string
	eventPublisher Publisher
}

// NatEventWaiter is responsible for waiting for nat events
type NatEventWaiter interface {
	WaitForEvent() event.Event
}

// PortSupplier provides port needed to run a service on
type PortSupplier interface {
	Acquire() (port.Port, error)
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// NewPinger returns Pinger instance
func NewPinger(waiter NatEventWaiter, proxy *NATProxy, previousStage string, publisher Publisher) NatPinger {
	target := make(chan *Params)
	cancel := make(chan struct{})
	stop := make(chan struct{})
	stopNATProxy := make(chan struct{})
	return &Pinger{
		pingTarget:     target,
		pingCancelled:  cancel,
		stop:           stop,
		stopNATProxy:   stopNATProxy,
		natEventWaiter: waiter,
		natProxy:       proxy,
		previousStage:  previousStage,
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
			log.Info().Msg("Stop pinger called")
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
func (p *Pinger) PingProvider(ip string, port int, consumerPort int, stop <-chan struct{}) error {
	log.Info().Msg("NAT pinging to provider")

	conn, err := p.getConnection(ip, port, consumerPort)
	if err != nil {
		return errors.Wrap(err, "failed to get connection")
	}

	go func() {
		err := p.ping(conn)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	time.Sleep(pingInterval * time.Millisecond)
	err = p.pingReceiver(conn, stop)
	if err != nil {
		return err
	}

	// send one last ping request to end hole punching procedure gracefully
	err = p.sendPingRequest(conn, 128)
	if err != nil {
		return errors.Wrap(err, "remote ping failed")
	}

	p.pingCancelled <- struct{}{}

	if consumerPort > 0 {
		consumerAddr := fmt.Sprintf("127.0.0.1:%d", consumerPort+1)
		log.Info().Msg("Handing connection to consumer NATProxy: " + consumerAddr)
		p.stopNATProxy = p.natProxy.consumerHandOff(consumerAddr, conn)
	}
	return nil
}

func (p *Pinger) ping(conn *net.UDPConn) error {
	// Windows detects that 1 TTL is too low and throws an exception during send
	ttl := 0
	i := 0

	for {
		select {
		case <-p.pingCancelled:
			return nil

		case <-time.After(pingInterval * time.Millisecond):
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

			if i*pingInterval > pingTimeout {
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
	timeout := time.After(pingTimeout * time.Millisecond)
	buf := make([]byte, bufferLen)
	for {
		select {
		case <-timeout:
			p.pingCancelled <- struct{}{}
			return errNATPunchAttemptTimedOut
		case <-stop:
			p.pingCancelled <- struct{}{}
			return errNATPunchAttemptStopped
		default:
		}

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

// SetProtectSocketCallback sets socket protection callback to be called when new socket is created in consumer NATProxy
func (p *Pinger) SetProtectSocketCallback(socketProtect func(socket int) bool) {
	p.natProxy.setProtectSocketCallback(socketProtect)
}

// StopNATProxy stops NATProxy launched by NATPinger
func (p *Pinger) StopNATProxy() {
	close(p.stopNATProxy)
}

// Valid returns that this pinger is a valid pinger
func (p *Pinger) Valid() bool {
	return true
}

func (p *Pinger) pingTargetConsumer(pingParams *Params) {
	log.Info().Msgf("Pinging peer with: %v", pingParams)

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

	go func() {
		err := p.ping(conn)
		if err != nil {
			log.Warn().Err(err).Msg("Error while pinging")
		}
	}()

	err = p.pingReceiver(conn, pingParams.Cancel)
	if err != nil {
		log.Error().Err(err).Msg("Ping receiver error")
		return
	}

	p.pingCancelled <- struct{}{}

	p.eventPublisher.Publish(event.Topic, event.BuildSuccessfulEvent(StageName))

	log.Info().Msg("Ping received, waiting for a new connection")

	go p.natProxy.handOff(pingParams.ProxyPortMappingKey, conn)
}
