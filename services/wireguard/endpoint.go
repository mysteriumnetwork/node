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

package wireguard

import (
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

// EndpointFactory creates new connection endpoint.
type EndpointFactory func() (ConnectionEndpoint, error)

// ConnectionEndpoint represents Wireguard network instance, it provide information
// required for establishing connection between service provider and consumer.
type ConnectionEndpoint interface {
	StartConsumerMode(config wgcfg.DeviceConfig) error
	ReconfigureConsumerMode(config wgcfg.DeviceConfig) error
	StartProviderMode(publicIP string, config wgcfg.DeviceConfig) error
	PeerStats() (wgcfg.Stats, error)
	Config() (ServiceConfig, error)
	InterfaceName() string
	Stop() error
	
	Diag() error
}
