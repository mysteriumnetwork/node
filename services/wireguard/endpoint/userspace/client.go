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
	"bufio"
	"fmt"
	"strings"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

type client struct {
	tun    tun.Device
	devAPI *device.Device
}

// NewWireguardClient creates new wireguard user space client.
func NewWireguardClient() (*client, error) {
	return &client{}, nil
}

func (c *client) ConfigureDevice(config wgcfg.DeviceConfig) (err error) {
	if c.tun, err = CreateTUN(config.IfaceName, config.Subnet); err != nil {
		return errors.Wrap(err, "failed to create TUN device")
	}

	c.devAPI = device.NewDevice(c.tun, device.NewLogger(device.LogLevelDebug, "[userspace-wg]"))
	if err := c.setDeviceConfig(config.Encode()); err != nil {
		return errors.Wrap(err, "failed to configure initial device")
	}

	// For consumer mode we need to exclude provider's IP from VPN tunnel
	// and add default routes to forward all traffic via VPN tunnel.
	if config.Peer.Endpoint != nil {
		if err := netutil.ExcludeRoute(config.Peer.Endpoint.IP); err != nil {
			return fmt.Errorf("could not exclude route %s: %w", config.Peer.Endpoint.IP.String(), err)
		}
		if err := netutil.AddDefaultRoute(config.IfaceName); err != nil {
			return fmt.Errorf("could not add default route for %s: %w", config.IfaceName, err)
		}
	}

	return nil
}
func (c *client) Close() error {
	c.devAPI.Close() // c.devAPI.Close() closes c.tun too
	return nil
}

func (c *client) PeerStats(string) (*wgcfg.Stats, error) {
	deviceState, err := ParseUserspaceDevice(c.devAPI.IpcGetOperation)
	if err != nil {
		return nil, err
	}
	stats, err := ParseDevicePeerStats(deviceState)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (c *client) DestroyDevice(name string) error {
	return destroyDevice(name)
}

func (c *client) setDeviceConfig(config string) error {
	if err := c.devAPI.IpcSetOperation(bufio.NewReader(strings.NewReader(config))); err != nil {
		return errors.Wrap(err, "failed to set device config")
	}
	c.devAPI.Up()
	return nil
}
