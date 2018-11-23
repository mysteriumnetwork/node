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

package network

// NewNetwork creates Wireguard client with predefined interface name and public IP.
func NewNetwork(name, publicIP string) (*network, error) {
	return nil, errNotSupported
}

// Provider starts and configure wireguard network interface for providing service.
// It returns information required to establish connection to the service.
func (n *network) Provider() (Provider, error) {
	return Provider{}, errNotSupported
}

// Consumer adds service consumer public key to the list of allowed peers.
// It returns information required to configure a consumer instance to allow connections.
func (n *network) Consumer() (Consumer, error) {
	return Consumer{}, errNotSupported
}

// Close closes wireguard client and destroys wireguard network interface.
func (n *network) Close() error {
	return errNotSupported
}
