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
	"encoding/base64"
	"errors"
	"net"

	"git.zx2c4.com/wireguard-go/device"
	"git.zx2c4.com/wireguard-go/tun"
	"github.com/mysteriumnetwork/node/consumer"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
)

type client struct {
	tun    tun.TUNDevice
	devAPI *device.DeviceApi
}

// NewWireguardClient creates new wireguard user space client.
func NewWireguardClient() (*client, error) {
	return &client{}, nil
}

func (c *client) ConfigureDevice(name string, config wg.DeviceConfig, subnet net.IPNet) (err error) {
	if c.tun, err = tun.CreateTUN(name, device.DefaultMTU); err != nil {
		return err
	}
	if err := assignIP(name, subnet); err != nil {
		return err
	}

	c.devAPI = device.UserspaceDeviceApi(c.tun)
	if err := c.devAPI.SetListeningPort(uint16(config.ListenPort())); err != nil {
		return err
	}

	key, err := base64stringTo32ByteArray(config.PrivateKey())
	if err != nil {
		return err
	}

	if err := c.devAPI.SetPrivateKey(device.NoisePrivateKey(key)); err != nil {
		return err
	}

	c.devAPI.Boot()
	return nil
}

func (c *client) AddPeer(name string, peer wg.PeerInfo) error {
	key, err := base64stringTo32ByteArray(peer.PublicKey())
	if err != nil {
		return err
	}

	extPeer := device.ExternalPeer{
		PublicKey:  device.NoisePublicKey(key),
		AllowedIPs: []string{"0.0.0.0/0"},
	}

	if ep := peer.Endpoint(); ep != nil {
		extPeer.RemoteEndpoint, err = device.CreateEndpoint(ep.String())
		if err != nil {
			return err
		}
	}

	return c.devAPI.AddPeer(extPeer)
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

func (c *client) PeerStats() (stats consumer.SessionStatistics, lastHandshake int, err error) {
	peers, err := c.devAPI.Peers()
	if err != nil {
		return consumer.SessionStatistics{}, 0, nil
	}

	if len(peers) != 1 {
		return consumer.SessionStatistics{}, 0, errors.New("exactly 1 peer expected")
	}

	return consumer.SessionStatistics{
		BytesSent:     peers[0].Stats.Sent,
		BytesReceived: peers[0].Stats.Received,
	}, peers[0].LastHanshake, nil
}

func (c *client) DestroyDevice(name string) error {
	return destroyDevice(name)
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
