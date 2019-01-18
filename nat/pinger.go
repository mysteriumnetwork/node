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

package nat

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/mysteriumnetwork/node/core/node"

	"github.com/mysteriumnetwork/node/core/service"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/pkg/errors"
)

const prefix = "[ NATPinger ] "

type Pinger struct {
	pingTarget    chan json.RawMessage
	pingReceived  chan bool
	pingCancelled chan bool
}

func NewPingerFactory(options node.Options) *Pinger {
	target := make(chan json.RawMessage)
	received := make(chan bool)
	cancel := make(chan bool)
	return &Pinger{
		pingTarget:    target,
		pingReceived:  received,
		pingCancelled: cancel,
	}
}

func (p *Pinger) Start() error {
	log.Info(prefix, "Starting a NAT pinger")
	for {
		select {
		case dst := <-p.pingTarget:
			conn, err := getConnection(dst)
			if err != nil {
				return err
			}

			go p.pingReceiver(conn)
			p.ping(conn)
			conn.Close()
		}
	}

	return nil
}

func getConnection(config json.RawMessage) (*net.UDPConn, error) {
	var c openvpn.ConsumerConfig
	err := json.Unmarshal(config, c)
	if err != nil {
		return nil, errors.Wrap(err, "parsing consumer address failed")
	}

	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.IP, c.Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", &net.UDPAddr{Port: 1194}, udpAddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *Pinger) ping(conn *net.UDPConn) error {
	for {
		select {
		case <-p.pingCancelled:
			return nil

		case <-time.After(500 * time.Millisecond):
			_, err := conn.Write([]byte("continuously pinging to " + conn.RemoteAddr().String()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Pinger) PingTargetChan() chan json.RawMessage {
	return p.pingTarget
}

func (p *Pinger) Bind(options service.Options) error {
	// TODO: parse service options and bind to localport
	return nil
}

func (p *Pinger) WaitForHole() error {
	_, ok := <-p.pingReceived
	if ok == false {
		return errors.New("NATPinger channel has been closed")
	}
	return nil
}

func (p *Pinger) pingReceiver(conn *net.UDPConn) {
	// TODO: get predefined service port or randomised one from above
	// TODO: how to get these addresses here, from here: func (openvpn *OpenvpnProcess) Start() error {

	for {
		var buf [512]byte
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			log.Errorf(prefix, "Failed to read remote peer: %s cause: %s", conn.RemoteAddr().String(), err)
		}
		p.pingReceived <- true
		p.pingCancelled <- true

		fmt.Println("remote peer address: ", addr)
		fmt.Println("remote peer PID: ", string(buf[:n]))
		daytime := time.Now().String()
		address := conn.LocalAddr().String()
		conn.WriteToUDP([]byte("Peer address: "+address), addr)
		conn.WriteToUDP([]byte(daytime), addr)
		return
	}
}
