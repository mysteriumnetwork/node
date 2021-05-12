/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package p2p

import (
	"fmt"
	"net"
	"sync"

	"github.com/pion/stun"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
)

const AppTopicSTUN = "STUN detection"

var serverList = []string{"stun.l.google.com:19302", "stun1.l.google.com:19302", "stun1.l.google.com:19302"}

func stunPorts(eventBus eventbus.EventBus, localPorts ...int) (remotePorts []int) {
	m := make(map[int]int)

	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(localPorts))

	for _, p := range localPorts {
		go func(p int) {
			defer wg.Done()

			resp := multiServerSTUN(serverList, p, 2)

			mu.Lock()
			defer mu.Unlock()

			for _, port := range resp {
				if m[p] != 0 {
					// TODO use correct naming for NAT type detection
					switch {
					case m[p] == port && p == port:
						eventBus.Publish(AppTopicSTUN, "full")
					case m[p] == port:
						eventBus.Publish(AppTopicSTUN, "semi")
					default:
						eventBus.Publish(AppTopicSTUN, "fail")
					}
				}

				m[p] = port
			}
		}(p)
	}

	wg.Wait()

	for _, p := range localPorts {
		remotePorts = append(remotePorts, m[p])
	}

	return remotePorts
}

func multiServerSTUN(servers []string, p, limit int) (respPort []int) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: p})
	if err != nil {
		log.Error().Err(err).Msg("failed to listen UDP address for STUN server")
		return nil
	}

	defer conn.Close()

	ch := make(chan int, len(servers))
	wg := sync.WaitGroup{}
	wg.Add(len(servers))

	go func() {
		wg.Wait()
		close(ch)
	}()

	for _, server := range servers {
		go func(server string) {
			defer wg.Done()

			port, err := stunPort(conn, server)
			if err != nil {
				log.Trace().Err(err).Msg("failed to get public UDP port from STUN server")
				return
			}

			ch <- port
		}(server)
	}

	for p := range ch {
		respPort = append(respPort, p)
		if len(respPort) == limit {
			return respPort
		}
	}

	return nil
}

func stunPort(conn *net.UDPConn, server string) (remotePort int, err error) {
	serverAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve STUN server address: %w", err)
	}

	m := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	_, err = conn.WriteToUDP(m.Raw, serverAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to send binding request to STUN server: %w", err)
	}

	msg := make([]byte, 1024)

	n, _, err := conn.ReadFromUDP(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to read message from STUN server: %w", err)
	}

	msg = msg[:n]

	switch {
	case stun.IsMessage(msg):
		m := &stun.Message{
			Raw: msg,
		}

		decErr := m.Decode()
		if decErr != nil {
			return 0, fmt.Errorf("failed to decode STUN server message: %w", err)
		}

		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(m); getErr != nil {
			return 0, fmt.Errorf("failed to decode STUN server message: %w", err)
		}

		return xorAddr.Port, nil

	default:
		log.Error().Msgf("unknown message: %s", msg)
	}

	return
}
