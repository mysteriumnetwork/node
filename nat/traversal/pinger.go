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
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
	"golang.org/x/net/ipv4"
)

const prefix = "[NATPinger] "
const pingInterval = 200
const pingTimeout = 10000

// ConfigParser is able to parse a config from given raw json
type ConfigParser interface {
	Parse(config json.RawMessage) (ip string, port int, err error)
}

// Pinger represents NAT pinger structure
type Pinger struct {
	localPort      int
	pingTarget     chan json.RawMessage
	pingReceived   chan struct{}
	pingCancelled  chan struct{}
	natEventWaiter NatEventWaiter
	configParser   ConfigParser
	stop           chan struct{}
	once           sync.Once
}

// NatEventWaiter is responsible for waiting for nat events
type NatEventWaiter interface {
	WaitForEvent() Event
}

// NewPingerFactory returns Pinger instance
func NewPingerFactory(waiter NatEventWaiter, parser ConfigParser) *Pinger {
	target := make(chan json.RawMessage)
	received := make(chan struct{})
	cancel := make(chan struct{})
	stop := make(chan struct{})
	return &Pinger{
		pingTarget:     target,
		pingReceived:   received,
		pingCancelled:  cancel,
		natEventWaiter: waiter,
		configParser:   parser,
		stop:           stop,
	}
}

// Start starts NAT pinger and waits for pingTarget to ping
func (p *Pinger) Start() {
	log.Info(prefix, "Starting a NAT pinger")
	for {
		select {
		case <-p.stop:
			return
		case dst := <-p.pingTarget:
			IP, port, err := p.configParser.Parse(dst)
			if err != nil {
				log.Warn(prefix, errors.Wrap(err, fmt.Sprintf("unable to parse ping message: %v", string(dst))))
			}

			log.Infof("%s ping target received: IP: %v, port: %v", prefix, IP, port)
			if port == 0 {
				// client did not sent its port to ping to, notifying the service to start
				p.pingReceived <- struct{}{}
				continue
			}
			conn, err := p.getConnection(IP, port)
			if err != nil {
				log.Error(prefix, "failed to get connection: ", err)
				continue
			}
			defer conn.Close()

			go func() {
				err := p.ping(conn)
				if err != nil {
					log.Warn(prefix, "Error while pinging: ", err)
				}
			}()

			select {
			case <-p.stop:
				return
			case <-time.After(pingInterval * time.Millisecond):
			}

			err = p.pingReceiver(conn)
			if err != nil {
				log.Error(prefix, "ping receiver error: ", err)
				continue
			}

			p.pingReceived <- struct{}{}
			log.Info(prefix, "ping received, waiting for a new connection")
			conn.Close()
		}
	}
}

// Stop stops the nat pinger
func (p *Pinger) Stop() {
	p.once.Do(func() { close(p.stop) })
}

// PingProvider pings provider determined by destination provided in sessionConfig
func (p *Pinger) PingProvider(ip string, port int) error {
	log.Info(prefix, "NAT pinging to provider")

	conn, err := p.getConnection(ip, port)
	if err != nil {
		return errors.Wrap(err, "failed to get connection")
	}
	defer conn.Close()

	//let provider ping first
	time.Sleep(20 * time.Millisecond)

	go func() {
		err := p.ping(conn)
		if err != nil {
			log.Warn(prefix, "Error while pinging: ", err)
		}
	}()

	time.Sleep(pingInterval * time.Millisecond)
	err = p.pingReceiver(conn)
	if err != nil {
		return err
	}

	// wait for provider to startup
	time.Sleep(3 * time.Second)

	return nil
}

func (p *Pinger) ping(conn *net.UDPConn) error {
	n := 1

	for {
		select {
		case <-p.pingCancelled:
			return nil

		case <-time.After(pingInterval * time.Millisecond):
			log.Trace(prefix, "pinging.. ")
			// This is the essence of the TTL based udp punching.
			// We're slowly increasing the TTL so that the packet is held.
			// After a few attempts we're setting the value to 128 and assuming we're through.
			if n > 4 {
				n = 128
			}

			err := ipv4.NewConn(conn).SetTTL(n)
			if err != nil {
				return errors.Wrap(err, "pinger setting ttl failed")
			}

			n++

			_, err = conn.Write([]byte("continuously pinging to " + conn.RemoteAddr().String()))
			if err != nil {
				return err
			}
		}
	}
}

func (p *Pinger) getConnection(ip string, port int) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	log.Info(prefix, "remote socket: ", udpAddr.String())

	conn, err := net.DialUDP("udp", &net.UDPAddr{Port: p.localPort}, udpAddr)
	if err != nil {
		return nil, err
	}

	log.Info(prefix, "local socket: ", conn.LocalAddr().String())

	return conn, nil
}

// PingTarget relays ping target address data
func (p *Pinger) PingTarget(target json.RawMessage) {
	p.pingTarget <- target
}

// BindPort gets port from session creation config and binds Pinger port to ping from
func (p *Pinger) BindPort(port int) {
	p.localPort = port
}

// WaitForHole waits while ping from remote peer is received
func (p *Pinger) WaitForHole() error {
	// TODO: check if three is a nat to punch
	events := make(chan Event)
	go func() {
		events <- p.natEventWaiter.WaitForEvent()
	}()

	select {
	case event := <-events:
		if event.Type == SuccessEventType {
			return nil
		}
		log.Info(prefix, "waiting for NAT pin-hole")
		_, ok := <-p.pingReceived
		if !ok {
			return errors.New("NATPinger channel has been closed")
		}
		return nil
	case <-p.stop:
		return errors.New("NAT wait cancelled")
	}
}

func (p *Pinger) pingReceiver(conn *net.UDPConn) error {
	timeout := time.After(pingTimeout * time.Millisecond)
	for {
		select {
		case <-p.stop:
			p.pingCancelled <- struct{}{}
			return errors.New("NAT punch attempt cancelled")
		case <-timeout:
			p.pingCancelled <- struct{}{}
			return errors.New("NAT punch attempt timed out")
		default:
		}

		var buf [512]byte
		n, err := conn.Read(buf[0:])
		if err != nil {
			log.Errorf(prefix, "Failed to read remote peer: %s cause: %s", conn.RemoteAddr().String(), err)
			time.Sleep(pingInterval * time.Millisecond)
			continue
		}
		fmt.Println("remote peer data received: ", string(buf[:n]))

		// send another couple of pings to remote side, because only now we have a pinghole
		// or wait for you pings to reach other end before closing pinger conn.
		select {
		case <-p.stop:
			p.pingCancelled <- struct{}{}
			return errors.New("NAT punch attempt cancelled")
		case <-time.After(2 * pingInterval * time.Millisecond):
			p.pingCancelled <- struct{}{}
			return nil
		}
	}
}
