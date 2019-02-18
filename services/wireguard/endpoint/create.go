// +build !linux linux,android

/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package endpoint

import (
	"github.com/mysteriumnetwork/node/core/location"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
)

// NewConnectionEndpoint creates new wireguard connection endpoint.
func NewConnectionEndpoint(
	location location.ServiceLocationInfo,
	resourceAllocator *resources.Allocator,
	portMap func(port int) (releasePortMapping func()),
	connectDelay int) (wg.ConnectionEndpoint, error) {

	client, err := userspace.NewWireguardClient()
	return &connectionEndpoint{
		wgClient:           client,
		location:           location,
		resourceAllocator:  resourceAllocator,
		mapPort:            portMap,
		releasePortMapping: func() {},
		connectDelay:       connectDelay,
	}, err
}
