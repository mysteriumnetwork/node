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
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
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

func (c *client) ConfigureDevice(name string, config wg.DeviceConfig, subnet net.IPNet) (err error) {
	if c.tun, err = CreateTUN(name, subnet); err != nil {
		return errors.Wrap(err, "failed to create TUN device")
	}

	c.devAPI = device.NewDevice(c.tun, device.NewLogger(device.LogLevelDebug, "[userspace-wg]"))
	if err := c.setDeviceConfig(config.Encode()); err != nil {
		return errors.Wrap(err, "failed to configure initial device")
	}
	return nil
}

func (c *client) AddPeer(iface string, peer wg.AddPeerOptions, _ ...string) error {
	p := wg.Peer{
		PublicKey:  peer.PublicKey,
		AllowedIPs: []string{"0.0.0.0/0", "::/0"},
		Endpoint:   peer.Endpoint,
	}
	if err := c.setDeviceConfig(p.Encode()); err != nil {
		errors.Wrap(err, "failed to add device peer")
	}
	return nil
}

func (c *client) RemovePeer(_ string, publicKey string) error {
	key, err := base64stringTo32ByteArray(publicKey)
	if err != nil {
		return err
	}

	c.devAPI.RemovePeer(key)
	return nil
}

func (c *client) Close() error {
	c.devAPI.Close() // c.devAPI.Close() closes c.tun too
	return nil
}

func (c *client) ConfigureRoutes(iface string, ip net.IP) error {
	if err := excludeRoute(ip); err != nil {
		return err
	}
	return addDefaultRoute(iface)
}

func (c *client) PeerStats() (*wg.Stats, error) {
	deviceState, err := wg.ParseUserspaceDevice(c.devAPI.IpcGetOperation)
	if err != nil {
		return nil, err
	}
	stats, err := wg.ParseDevicePeerStats(deviceState)
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
	fmt.Printf("---------setDeviceConfig: %s\n", config)
	return nil
}

func base64stringTo32ByteArray(s string) (res [32]byte, err error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return res, err
	} else if len(decoded) != 32 {
		return res, errors.New("unexpected key size")
	}

	copy(res[:], decoded)
	return
}
