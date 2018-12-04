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

package kernelspace

import (
	"encoding/base64"
	"net"

	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
)

var allowedIPs = []net.IPNet{
	{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
}

type client struct {
	wgClient *wireguardctrl.Client
}

// NewWireguardClient creates new wireguard kernel space client.
func NewWireguardClient() (*client, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}
	return &client{wgClient}, nil
}

func (c *client) ConfigureDevice(name string, config wg.Device) error {
	var deviceConfig wgtypes.Config

	if port := config.ListenPort(); port != nil {
		deviceConfig.ListenPort = port
	}

	if key := config.PrivateKey(); key != nil {
		k, err := base64.StdEncoding.DecodeString(*key)
		if err != nil {
			return err
		}
		privateKey, err := wgtypes.NewKey(k)
		if err != nil {
			return err
		}
		deviceConfig.PrivateKey = &privateKey
	}

	var peerPublicKey *wgtypes.Key
	if key := config.PeerPublicKey(); key != nil {
		k, err := base64.StdEncoding.DecodeString(*key)
		if err != nil {
			return err
		}
		publicKey, err := wgtypes.NewKey(k)
		if err != nil {
			return err
		}
		peerPublicKey = &publicKey
	}
	endpoint := config.PeerEndpoint()

	if endpoint != nil || peerPublicKey != nil {
		deviceConfig.Peers = []wgtypes.PeerConfig{{
			Endpoint:   endpoint,
			PublicKey:  *peerPublicKey,
			AllowedIPs: allowedIPs,
		}}
	}

	return c.wgClient.ConfigureDevice(name, deviceConfig)
}

func (c *client) Device(name string) (wg.Device, error) {
	d, err := c.wgClient.Device(name)
	if err != nil || d.Name != name {
		return device{}, err
	}

	return device{
		name:       name,
		privateKey: d.PrivateKey,
	}, nil
}

func (c *client) Close() error {
	return c.wgClient.Close()
}

type device struct {
	name       string
	privateKey wgtypes.Key
}

func (d device) Name() *string {
	return &d.name
}

func (d device) PrivateKey() *string {
	key := d.privateKey.String()
	return &key
}

func (d device) PublicKey() *string {
	key := d.privateKey.PublicKey().String()
	return &key
}

func (d device) ListenPort() *int           { return nil }
func (d device) PeerEndpoint() *net.UDPAddr { return nil }
func (d device) PeerPublicKey() *string     { return nil }
