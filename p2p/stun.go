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
	"time"

	"github.com/pion/stun"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests/resolver"
)

// AppTopicSTUN represents the STUN detection topic.
const AppTopicSTUN = "STUN detection"

// STUNDetectionStatus represents information about detected NAT type using STUN servers.
type STUNDetectionStatus struct {
	Identity string
	NATType  string
}

func stunPorts(identity identity.Identity, eventBus eventbus.Publisher, localPorts ...int) (remotePorts []int) {
	serverList := config.GetStringSlice(config.FlagSTUNservers)
	if len(serverList) == 0 {
		return localPorts
	}

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

			natType := "unknown"

			for _, port := range resp {
				if m[p] != 0 {
					switch {
					case m[p] == port && p == port:
						natType = "full"
					case m[p] == port:
						natType = "restricted"
					default:
						natType = "fail"
					}
				}

				if port != 0 {
					m[p] = port
				}
			}

			if natType == "fail" {
				delete(m, p)
			}

			if eventBus != nil {
				eventBus.Publish(AppTopicSTUN, STUNDetectionStatus{
					Identity: identity.Address,
					NATType:  natType,
				})
			}
		}(p)
	}

	wg.Wait()

	for _, p := range localPorts {
		if port, ok := m[p]; ok {
			remotePorts = append(remotePorts, port)
		} else {
			remotePorts = append(remotePorts, p)
		}
	}

	return remotePorts
}

func multiServerSTUN(servers []string, p, limit int) (respPort []int) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: p})
	if err != nil {
		log.Error().Err(err).Msg("failed to listen UDP address for STUN server")
		return nil
	}

	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		log.Error().Err(err).Msg("failed to set connection deadline for STUN server")
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

	return respPort
}

func stunPort(conn *net.UDPConn, server string) (remotePort int, err error) {
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		return 0, fmt.Errorf("failed to parse STUN server address: %w", err)
	}

	if addrs := resolver.FetchDNSFromCache(host); len(addrs) > 0 {
		server = net.JoinHostPort(addrs[0], port)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve STUN server address: %w", err)
	}

	resolver.CacheDNSRecord(host, []string{serverAddr.IP.String()})

	m := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if _, err = conn.WriteToUDP(m.Raw, serverAddr); err != nil {
		return 0, fmt.Errorf("failed to send binding request to STUN server: %w", err)
	}

	msg := make([]byte, 1024)

	n, _, err := conn.ReadFromUDP(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to read message from STUN server: %w", err)
	}

	msg = msg[:n]

	if !stun.IsMessage(msg) {
		return 0, fmt.Errorf("not correct response from STUN server")
	}

	resp := &stun.Message{Raw: msg}

	if err := resp.Decode(); err != nil {
		return 0, fmt.Errorf("failed to decode STUN server message: %w", err)
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(resp); err != nil {
		return 0, fmt.Errorf("failed to decode STUN server message: %w", err)
	}

	return xorAddr.Port, nil
}
