/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package socks5

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/market"
)

// Bootstrap is called on program initialization time and registers various deserializers related to SOCKS5 service
func Bootstrap() {
	market.RegisterServiceDefinitionUnserializer(
		ServiceType,
		func(rawDefinition *json.RawMessage) (market.ServiceDefinition, error) {
			var definition ServiceDefinition
			err := json.Unmarshal(*rawDefinition, &definition)

			return definition, err
		},
	)
}

// ServiceType indicates "socks5" service type
const ServiceType = "socks5"

// ServiceDefinition structure represents "socks5" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location market.Location `json:"location"`

	// Approximate information on location where the actual tunnelled traffic will originate from.
	// This is used by providers having their own means of setting tunnels to other remote exit points.
	LocationOriginate market.Location `json:"location_originate"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() market.Location {
	return service.Location
}
