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

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/net/ipv4"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/pkg/errors"
)

const prefix = "[ NATPinger ] "
const pingInterval = 100

type Pinger struct {
	localPort     int
	pingTarget    chan json.RawMessage
	pingReceived  chan bool
	pingCancelled chan bool
}

func NewPingerFactory() *Pinger {
	target := make(chan json.RawMessage)
	received := make(chan bool)
	cancel := make(chan bool)
	return &Pinger{
		pingTarget:    target,
		pingReceived:  received,
		pingCancelled: cancel,
	}
}

func (p *Pinger) Start() {
	log.Info(prefix, "Starting a NAT pinger")
	for {
		select {
		case dst := <-p.pingTarget:
			IP, port, err := parseConsumerConfig(dst)
			conn, err := p.getConnection(IP, port)
			if err != nil {
				log.Error(errors.Wrap(err, "failed to get connection"))
				return
			}
			go p.ping(conn)
			time.Sleep(pingInterval * time.Millisecond)
			p.pingReceiverPlain(conn)
			p.pingReceived <- true
			log.Info(prefix, "ping received, waiting for a new connection")
			conn.Close()
		}
	}

	return
}

func (p *Pinger) PingProvider(sessionConfig json.RawMessage) error {
	log.Info(prefix, "NAT pinging to provider")

	spew.Dump(sessionConfig)
	ip, port, err := parseProviderConfig(sessionConfig)
	conn, err := p.getConnection(ip, port)
	if err != nil {
		return errors.Wrap(err, "failed to get connection")
	}

	go p.ping(conn)
	time.Sleep(pingInterval * time.Millisecond)
	p.pingReceiverPlain(conn)
	// wait for provider to startup
	time.Sleep(4 * time.Second)
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

func parseConsumerConfig(config json.RawMessage) (IP string, port int, err error) {
	var c openvpn.ConsumerConfig
	err = json.Unmarshal(config, &c)
	if err != nil {
		return "", 0, errors.Wrap(err, "parsing consumer address:port failed")
	}
	return c.IP, c.Port, nil
}

func parseProviderConfig(sessionConfig json.RawMessage) (IP string, port int, err error) {
	var vpnConfig openvpn.VPNConfig
	err = json.Unmarshal(sessionConfig, &vpnConfig)
	log.Infof(prefix, "Provider Config: %v", vpnConfig)
	spew.Dump(vpnConfig)
	if err != nil {
		return "", 0, errors.Wrap(err, "parsing provider address:port failed")
	}
	return vpnConfig.RemoteIP, vpnConfig.RemotePort, nil
}

func (p *Pinger) getConnection(ip string, port int) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	log.Info(prefix, "remote socket: ", udpAddr.String())

	conn, err := net.DialUDP("udp", &net.UDPAddr{Port: p.localPort}, udpAddr)

	log.Info(prefix, "local socket: ", conn.LocalAddr().String())

	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *Pinger) PingTargetChan() chan json.RawMessage {
	return p.pingTarget
}

func (p *Pinger) BindProducer(port int) error {
	/*
		// TODO: parse service options and bind to localport
		openvpnConfig, ok := options.Options.(openvpn_service.Options)
		if ok {
			p.localPort = openvpnConfig.OpenvpnPort
		} else {
			return errors.New("Failed to use producer config")
		}
	*/
	p.localPort = port
	return nil
}

func (p *Pinger) BindConsumer(config connection.ConsumerConfig) error {
	openvpnConfig, ok := config.(*openvpn.ConsumerConfig)
	if ok {
		p.localPort = openvpnConfig.Port
	} else {
		return errors.New("Failed to use consumer config")
	}
	return nil
}

func (p *Pinger) WaitForHole() error {
	_, ok := <-p.pingReceived
	if ok == false {
		return errors.New("NATPinger channel has been closed")
	}
	return nil
}

func (p *Pinger) pingReceiverPlain(conn *net.UDPConn) {
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
