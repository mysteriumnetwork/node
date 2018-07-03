/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dto

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

// ServiceDefinition structure represents various service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location dto_discovery.Location `json:"location"`

	// Approximate information on location where the tunnelled traffic will originate from
	LocationOriginate dto_discovery.Location `json:"location_originate"`

	// Available per session bandwidth
	SessionBandwidth Bandwidth `json:"session_bandwidth,omitempty"`

	// Transport protocol used by service
	Protocol string `json:"protocol,omitempty"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() dto_discovery.Location {
	return service.Location
}
