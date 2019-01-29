package nat

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

import (
	"time"

	log "github.com/cihub/seelog"
	portmap "github.com/ethereum/go-ethereum/p2p/nat"
)

const logPrefix = "[nat] "

const (
	mapTimeout        = 20 * time.Minute
	mapUpdateInterval = 15 * time.Minute
)

// PortMapping maps given port of given protocol from external IP on a gateway to local machine internal IP
//  'name' denotes rule name added on a gateway
func PortMapping(protocol string, port int, name string) chan struct{} {
	mapperQuit := make(chan struct{})
	go mapPort(portmap.Any(),
		mapperQuit,
		protocol,
		port,
		port,
		name)

	return mapperQuit
}

// mapPort adds a port mapping on m and keeps it alive until c is closed.
// This function is typically invoked in its own goroutine.
func mapPort(m portmap.Interface, c chan struct{}, protocol string, extPort, intPort int, name string) {
	refresh := time.NewTimer(mapUpdateInterval)
	defer func() {
		refresh.Stop()
		log.Debug(logPrefix, "Deleting port mapping")
		m.DeleteMapping(protocol, extPort, intPort)
	}()
	if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
		log.Debugf("%s, Couldn't add port mapping: %v, %s", logPrefix, err, "retrying with permanent lease")
		if err := m.AddMapping(protocol, extPort, intPort, name, 0); err != nil {
			// some gateways support only permanent leases
			log.Debug(logPrefix, "Couldn't add port mapping: ", err)
		} else {
			log.Info(logPrefix, "Mapped network port: ", extPort)
		}
	} else {
		log.Info(logPrefix, "Mapped network port")
	}
	for {
		select {
		case _, ok := <-c:
			if !ok {
				return
			}
		case <-refresh.C:
			log.Trace(logPrefix, "Refreshing port mapping")
			if err := m.AddMapping(protocol, extPort, intPort, name, mapTimeout); err != nil {
				log.Debugf("%s, Couldn't add port mapping: %v, %s", logPrefix, err, "retrying with permanent lease")
				if err := m.AddMapping(protocol, extPort, intPort, name, 0); err != nil {
					// some gateways support only permanent leases
					log.Debug(logPrefix, "Couldn't add port mapping: ", err)
				}
			}
			refresh.Reset(mapUpdateInterval)
		}
	}
}
