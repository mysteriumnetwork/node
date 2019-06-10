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

package discovery

import (
	"sync"
)

// LANDiscovery provides local network discovery service for Mysterium Node UI.
type LANDiscovery interface {
	Start() error
	Stop() error
}

type multiDiscovery struct {
	ssdp    LANDiscovery
	bonjour LANDiscovery
	lock    sync.Mutex
}

// NewLANDiscoveryService creates SSDP and Bonjour services for LAN discovery.
func NewLANDiscoveryService(uiPort int) *multiDiscovery {
	return &multiDiscovery{
		ssdp:    newSSDPServer(uiPort),
		bonjour: newBonjourServer(uiPort),
	}
}

func (md *multiDiscovery) Start() error {
	if err := md.bonjour.Start(); err != nil {
		return err
	}

	return md.ssdp.Start()
}

func (md *multiDiscovery) Stop() error {
	// bonjour Stop does not return any error, nothing to check
	_ = md.bonjour.Stop()
	return md.ssdp.Stop()
}
