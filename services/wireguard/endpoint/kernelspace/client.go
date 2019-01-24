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
	"errors"
	"net"

	log "github.com/cihub/seelog"
	"github.com/jackpal/gateway"
	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/mysteriumnetwork/node/consumer"
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
	endpoint := peer.Endpoint()
	publicKey, err := stringToKey(peer.PublicKey())
	if err != nil {
		return err
	}

	var deviceConfig wgtypes.Config
	deviceConfig.Peers = []wgtypes.PeerConfig{{
		Endpoint:   endpoint,
		PublicKey:  publicKey,
		AllowedIPs: allowedIPs,
	}}
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) PeerStats() (stats consumer.SessionStatistics, lastHandshake int, err error) {
	d, err := c.wgClient.Device(c.iface)
	if err != nil {
		return consumer.SessionStatistics{}, 0, err
	}

	if len(d.Peers) != 1 {
		return consumer.SessionStatistics{}, 0, errors.New("exactly 1 peer expected")
	}

	return consumer.SessionStatistics{
		BytesReceived: uint64(d.Peers[0].ReceiveBytes),
		BytesSent:     uint64(d.Peers[0].TransmitBytes),
	}, int(d.Peers[0].LastHandshakeTime.Unix()), nil
}

func (c *client) DestroyDevice(name string) error {
	return utils.SudoExec("ip", "link", "del", "dev", name)
}

func (c *client) up(iface string, ipAddr net.IPNet) error {
	if d, err := c.wgClient.Device(iface); err != nil || d.Name != iface {
		if err := utils.SudoExec("ip", "link", "add", "dev", iface, "type", "wireguard"); err != nil {
			return err
		}
	}

	if err := utils.SudoExec("ip", "address", "replace", "dev", iface, ipAddr.String()); err != nil {
		return err
	}

	return utils.SudoExec("ip", "link", "set", "dev", iface, "up")
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

	return utils.SudoExec("ip", "route", "replace", ip.String(), "via", gw.String())
}

func addDefaultRoute(iface string) error {
	if err := utils.SudoExec("ip", "route", "replace", "0.0.0.0/1", "dev", iface); err != nil {
		return err
	}
	return utils.SudoExec("ip", "route", "replace", "128.0.0.0/1", "dev", iface)
}

func (c *client) Close() (err error) {
	var errs []error
	defer func() {
		for i := range errs {
			log.Error("failed to close wireguard kernelspace client: ", errs[i])
			if err == nil {
				err = errs[i]
			}
		}
	}()

	if err := c.DestroyDevice(c.iface); err != nil {
		errs = append(errs, err)
	}

	if err := c.wgClient.Close(); err != nil {
		errs = append(errs, err)
	}

	return nil
}

func stringToKey(key string) (wgtypes.Key, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.NewKey(k)
}
