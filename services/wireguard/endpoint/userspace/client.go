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
	"net"
	"time"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/wireguard-go/device"
	"github.com/mysteriumnetwork/wireguard-go/tun"
	"github.com/pkg/errors"
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
	if c.tun, err = CreateTUN(name, subnet); err != nil {
		return errors.Wrap(err, "failed to create TUN device")
	}

	c.devAPI = device.UserspaceDeviceApi(c.tun)
	if err := c.devAPI.SetListeningPort(uint16(config.ListenPort())); err != nil {
		return errors.Wrap(err, "failed to set listening port")
	}

	key, err := base64stringTo32ByteArray(config.PrivateKey())
	if err != nil {
		return errors.Wrap(err, "failed to parse private key from config")
	}

	if err := c.devAPI.SetPrivateKey(device.NoisePrivateKey(key)); err != nil {
		return errors.Wrap(err, "failed to set private key to userspace device API")
	}

	c.devAPI.Boot()
	return nil
}

func (c *client) AddPeer(name string, peer wg.PeerInfo, allowedIPs ...string) error {
	key, err := base64stringTo32ByteArray(peer.PublicKey())
	if err != nil {
		return err
	}

	extPeer := device.ExternalPeer{
		PublicKey:  device.NoisePublicKey(key),
		AllowedIPs: []string{"0.0.0.0/0", "::/0"},
	}

	if len(allowedIPs) > 0 {
		extPeer.AllowedIPs = allowedIPs
	}

	if ep := peer.Endpoint(); ep != nil {
		extPeer.RemoteEndpoint, err = device.CreateEndpoint(ep.String())
		if err != nil {
			return err
		}
	}

	return c.devAPI.AddPeer(extPeer)
}

func (c *client) DelPeer(_ string, publicKey string) error {
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

func (c *client) PeerStats() (wg.Stats, error) {
	peers, err := c.devAPI.Peers()
	if err != nil {
		return wg.Stats{}, nil
	}

	if len(peers) != 1 {
		return wg.Stats{}, errors.New("exactly 1 peer expected")
	}

	return wg.Stats{
		BytesSent:     peers[0].Stats.Sent,
		BytesReceived: peers[0].Stats.Received,
		LastHandshake: time.Unix(int64(peers[0].LastHanshake), 0),
	}, nil
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
