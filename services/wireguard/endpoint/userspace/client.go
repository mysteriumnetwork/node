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

package userspace

import (
	"errors"
	"net"

	"github.com/mysteriumnetwork/node/consumer"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
)

type client struct{}

// NewWireguardClient creates new wireguard user space client.
func NewWireguardClient() (*client, error) {
	return nil, errors.New("Not implemented")
}

func (c *client) ConfigureRoutes(iface string, ip net.IP) error { return nil }
func (c *client) DestroyDevice(name string) error               { return nil }
func (c *client) AddPeer(name string, peer wg.PeerInfo) error   { return nil }
func (c *client) Close() error                                  { return nil }
func (c *client) PeerStats() (consumer.SessionStatistics, error) {
	return consumer.SessionStatistics{}, nil
}
func (c *client) ConfigureDevice(name string, config wg.DeviceConfig, subnet net.IPNet) error {
	return nil
}
