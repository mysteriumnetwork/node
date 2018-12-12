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

	"github.com/jackpal/gateway"
	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/utils"
)

var allowedIPs = []net.IPNet{
	{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
}

type client struct {
	iface    string
	wgClient *wireguardctrl.Client
}

// NewWireguardClient creates new wireguard kernel space client.
func NewWireguardClient() (*client, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}
	return &client{wgClient: wgClient}, nil
}

func (c *client) ConfigureDevice(iface string, config wg.DeviceConfig, ipAddr net.IPNet) error {
	var deviceConfig wgtypes.Config
	if config != nil {
		port := config.ListenPort()
		privateKey, err := stringToKey(config.PrivateKey())
		if err != nil {
			return err
		}
		deviceConfig.PrivateKey = &privateKey
		deviceConfig.ListenPort = &port
	}

	if err := c.up(iface, ipAddr); err != nil {
		return err
	}
	c.iface = iface
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) AddPeer(iface string, peer wg.PeerInfo) error {
	var deviceConfig wgtypes.Config
	if peer != nil {
		endpoint := peer.Endpoint()
		publicKey, err := stringToKey(peer.PublicKey())
		if err != nil {
			return err
		}

		deviceConfig.Peers = []wgtypes.PeerConfig{{
			Endpoint:   endpoint,
			PublicKey:  publicKey,
			AllowedIPs: allowedIPs,
		}}
	}
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) up(iface string, ipAddr net.IPNet) error {
	if d, err := c.wgClient.Device(iface); err != nil || d.Name != iface {
		if err := utils.SuExec("ip", "link", "add", "dev", iface, "type", "wireguard"); err != nil {
			return err
		}
	}

	if err := utils.SuExec("ip", "address", "replace", "dev", iface, ipAddr.String()); err != nil {
		return err
	}

	return utils.SuExec("ip", "link", "set", "dev", iface, "up")
}

func (c *client) ConfigureRoutes(iface string, ip net.IP) error {
	if err := excludeRoute(ip); err != nil {
		return err
	}
	return addDefaultRoute(iface)
}

func excludeRoute(ip net.IP) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return err
	}

	return utils.SuExec("ip", "route", "replace", ip.String(), "via", gw.String())
}

func addDefaultRoute(iface string) error {
	if err := utils.SuExec("ip", "route", "replace", "0.0.0.0/1", "dev", iface); err != nil {
		return err
	}
	return utils.SuExec("ip", "route", "replace", "128.0.0.0/1", "dev", iface)
}

func (c *client) Close() error {
	d, err := c.wgClient.Device(c.iface)
	if err != nil || d.Name != c.iface {
		return err
	}

	if len(d.Peers) > 0 && d.Peers[0].Endpoint != nil {
		if err := utils.SuExec("ip", "route", "del", d.Peers[0].Endpoint.IP.String()); err != nil {
			return err
		}
	}

	if err := utils.SuExec("ip", "link", "del", "dev", c.iface); err != nil {
		return err
	}
	return c.wgClient.Close()
}

// GeneratePrivateKey creates new wireguard private key
func GeneratePrivateKey() (string, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", err
	}
	return key.String(), nil
}

// PrivateKeyToPublicKey generates wireguard public key from private key
func PrivateKeyToPublicKey(key string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	privateKey, err := wgtypes.NewKey(k)
	if err != nil {
		return "", err
	}
	return privateKey.PublicKey().String(), nil
}

func stringToKey(key string) (wgtypes.Key, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.NewKey(k)
}
