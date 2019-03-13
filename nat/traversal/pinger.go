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
	"time"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
	"golang.org/x/net/ipv4"
)

const prefix = "[NATPinger] "
const pingInterval = 200

type ConfigParser interface {
	Parse(config json.RawMessage) (ip string, port int, err error)
}

// Pinger represents NAT pinger structure
type Pinger struct {
	localPort      int
	pingTarget     chan json.RawMessage
	pingReceived   chan bool
	pingCancelled  chan bool
	natEventWaiter NatEventWaiter
	configParser   ConfigParser
}

// NatEventWaiter is responsible for waiting for nat events
type NatEventWaiter interface {
	WaitForEvent() Event
}

// NewPingerFactory returns Pinger instance
func NewPingerFactory(waiter NatEventWaiter, parser ConfigParser) *Pinger {
	target := make(chan json.RawMessage)
	received := make(chan bool)
	cancel := make(chan bool)
	return &Pinger{
		pingTarget:     target,
		pingReceived:   received,
		pingCancelled:  cancel,
		natEventWaiter: waiter,
		configParser:   parser,
	}
}

// Start starts NAT pinger and waits for pingTarget to ping
func (p *Pinger) Start() {
	log.Info(prefix, "Starting a NAT pinger")
	for {
		select {
		case dst := <-p.pingTarget:
			IP, port, err := p.configParser.Parse(dst)
			log.Infof("%s ping target received: IP: %v, port: %v", prefix, IP, port)
			if port == 0 {
				// client did not sent its port to ping to, attempting with service start
				p.pingReceived <- true
				continue
			}
			conn, err := p.getConnection(IP, port)
			if err != nil {
				log.Error(errors.Wrap(err, "failed to get connection"))
				continue
			}
			go p.ping(conn)
			time.Sleep(pingInterval * time.Millisecond)
			p.pingReceiver(conn)
			p.pingReceived <- true
			log.Info(prefix, "ping received, waiting for a new connection")
			conn.Close()
		}
	}

	return
}

// PingProvider pings provider determined by destination provided in sessionConfig
func (p *Pinger) PingProvider(ip string, port int) error {
	log.Info(prefix, "NAT pinging to provider")

	conn, err := p.getConnection(ip, port)
	if err != nil {
		return errors.Wrap(err, "failed to get connection")
	}

	//let provider ping first
	time.Sleep(20 * time.Millisecond)
	go p.ping(conn)
	time.Sleep(pingInterval * time.Millisecond)
	p.pingReceiver(conn)
	// wait for provider to startup
	time.Sleep(3 * time.Second)
	conn.Close()

	return nil
}

func (p *Pinger) ping(conn *net.UDPConn) error {
	n := 1

	for {
		select {
		case <-p.pingCancelled:
			return nil

		case <-time.After(pingInterval * time.Millisecond):
			log.Info(prefix, "pinging.. ")
			if n > 4 {
				n = 128
			}

			ipv4.NewConn(conn).SetTTL(n)
			n++

			_, err := conn.Write([]byte("continuously pinging to " + conn.RemoteAddr().String()))
			if err != nil {
				return err
			}
		}
	}

	return nil
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

// PingTargetChan returns a channel which relays ping target address data
func (p *Pinger) PingTargetChan() chan json.RawMessage {
	return p.pingTarget
}

// BindProvider binds local port from which Pinger will start its pinging
func (p *Pinger) BindProvider(port int) {
	log.Info(prefix, "provided bind port: ", port)
	p.localPort = port
}

// BindPort gets port from session creation config and binds Pinger port to ping from
func (p *Pinger) BindPort(port int) {
	p.localPort = port
}

// WaitForHole waits while ping from remote peer is received
func (p *Pinger) WaitForHole() error {
	// TODO: check if three is a nat to punch
	if p.natEventWaiter.WaitForEvent() == EventSuccess {
		return nil
	}
	log.Info(prefix, "waiting for NAT pin-hole")
	_, ok := <-p.pingReceived
	if ok == false {
		return errors.New("NATPinger channel has been closed")
	}
	return nil
}

func (p *Pinger) pingReceiver(conn *net.UDPConn) {
	for {
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
		time.Sleep(2 * pingInterval * time.Millisecond)

		p.pingCancelled <- true

		return
	}
}
