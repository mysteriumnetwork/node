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
	"net"
)

// EndpointFactory creates new connection endpoint.
type EndpointFactory func() (ConnectionEndpoint, error)

// ConnectionEndpoint represents Wireguard network instance, it provide information
// required for establishing connection between service provider and consumer.
type ConnectionEndpoint interface {
	StartConsumerMode(config ConsumerModeConfig) error
	StartProviderMode(config ProviderModeConfig) error
	PeerStats() (*Stats, error)
	Config() (ServiceConfig, error)
	InterfaceName() string
	Stop() error
}

// ConsumerModeConfig is consumer endpoint startup configuration.
type ConsumerModeConfig struct {
	PrivateKey string
	IPAddress  net.IPNet
	ListenPort int

	Peer Peer
}

// ProviderModeConfig is provider endpoint startup configuration.
type ProviderModeConfig struct {
	Network    net.IPNet
	ListenPort int
	PublicIP   string

	Peer Peer
}
