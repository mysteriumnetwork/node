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

	log "github.com/cihub/seelog"
	portmap "github.com/ethereum/go-ethereum/p2p/nat"
)

const logPrefix = "[port mapping] "

const (
	mapTimeout        = 20 * time.Minute
	mapUpdateInterval = 15 * time.Minute
)

// GetPortMappingFunc returns PortMapping function if service is behind NAT
func GetPortMappingFunc(pubIP, outIP, protocol string, port int, description string) func() {
	if pubIP != outIP {
		return PortMapping(protocol, port, description)
	}
	return func() {}
}

// PortMapping maps given port of given protocol from external IP on a gateway to local machine internal IP
// 'name' denotes rule name added on a gateway.
func PortMapping(protocol string, port int, name string) func() {
	mapperQuit := make(chan struct{})
	go mapPort(portmap.Any(),
		mapperQuit,
		protocol,
		port,
		port,
		name)

	return func() { close(mapperQuit) }
}

// mapPort adds a port mapping on m and keeps it alive until c is closed.
// This function is typically invoked in its own goroutine.
func mapPort(m portmap.Interface, c chan struct{}, protocol string, extPort, intPort int, name string) {
	defer func() {
		log.Debug(logPrefix, "Deleting port mapping for port: ", extPort)

		if err := m.DeleteMapping(protocol, extPort, intPort); err != nil {
			log.Debug(logPrefix, "Couldn't delete port mapping: ", err)
		}
	}()
	for {
		addMapping(m, protocol, extPort, intPort, name)
		select {
		case <-c:
			return
		case <-time.After(mapUpdateInterval):
		}
	}
}

func addMapping(m portmap.Interface, protocol string, extPort, intPort int, name string) {
	if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
		log.Debugf("%s, Couldn't add port mapping for port %d: %v, retrying with permanent lease", logPrefix, extPort, err)
		if err := m.AddMapping(protocol, extPort, intPort, name, 0); err != nil {
			// some gateways support only permanent leases
			log.Debugf("%s Couldn't add port mapping for port %d: %v", logPrefix, extPort, err)
			return
		}
	}
	log.Info(logPrefix, "Mapped network port:", extPort)
}
