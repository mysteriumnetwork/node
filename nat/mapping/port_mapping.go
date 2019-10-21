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

package mapping

import (
	"time"

	portmap "github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/rs/zerolog/log"
)

const (
	mapTimeout        = 20 * time.Minute
	mapUpdateInterval = 15 * time.Minute
)

// StageName is used to indicate port mapping NAT traversal stage
const StageName = "port_mapping"

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// GetPortMappingFunc returns PortMapping function if service is behind NAT
func GetPortMappingFunc(pubIP, outIP, protocol string, port int, description string, publisher Publisher) func() {
	if pubIP != outIP {
		return PortMapping(protocol, port, description, publisher)
	}
	return func() {}
}

// PortMapping maps given port of given protocol from external IP on a gateway to local machine internal IP
// 'name' denotes rule name added on a gateway.
func PortMapping(protocol string, port int, name string, publisher Publisher) func() {
	mapperQuit := make(chan struct{})
	go mapPort(portmap.Any(),
		mapperQuit,
		protocol,
		port,
		port,
		name,
		publisher)

	return func() { close(mapperQuit) }
}

// mapPort adds a port mapping on m and keeps it alive until c is closed.
// This function is typically invoked in its own goroutine.
func mapPort(m portmap.Interface, c chan struct{}, protocol string, extPort, intPort int, name string, publisher Publisher) {
	defer func() {
		log.Debug().Msgf("Deleting port mapping for port: %d", extPort)

		if err := m.DeleteMapping(protocol, extPort, intPort); err != nil {
			log.Warn().Err(err).Msg("Couldn't delete port mapping")
		}
	}()
	for {
		addMapping(m, protocol, extPort, intPort, name, publisher)
		select {
		case <-c:
			return
		case <-time.After(mapUpdateInterval):
		}
	}
}

func addMapping(m portmap.Interface, protocol string, extPort, intPort int, name string, publisher Publisher) {
	if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
		log.Warn().Err(err).Msgf("Couldn't add port mapping for port %d: retrying with permanent lease", extPort)
		if err := m.AddMapping(protocol, extPort, intPort, name, 0); err != nil {
			// some gateways support only permanent leases
			publisher.Publish(event.Topic, event.BuildFailureEvent(StageName, err))
			log.Warn().Err(err).Msgf("Couldn't add port mapping for port %d", extPort)
			return
		}
	}
	publisher.Publish(event.Topic, event.BuildSuccessfulEvent(StageName))
	log.Info().Msgf("Mapped network port: %d", extPort)
}
