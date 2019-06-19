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

package upnp

import (
	"sync"
)

// GatewayLoader fetches the gateways once and keeps them stored for further use
type GatewayLoader struct {
	gateways []GatewayDevice
	once     sync.Once
}

// HumanReadable returns a human readable representation of the devices
func (gl *GatewayLoader) HumanReadable() []map[string]string {
	gl.once.Do(gl.load)
	res := make([]map[string]string, 0)

	for _, v := range gl.gateways {
		res = append(res, v.ToMap())
	}

	return res
}

// Get returns the gateway devices
func (gl *GatewayLoader) Get() []GatewayDevice {
	gl.once.Do(gl.load)
	return gl.gateways
}

func (gl *GatewayLoader) load() {
	gateways, err := discoverGateways()
	if err != nil {
		log.Error("error discovering UPnP devices: ", err)
		return
	}
	gl.gateways = gateways
	printGatewayInfo(gateways)
}
