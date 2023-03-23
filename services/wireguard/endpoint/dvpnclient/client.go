/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package dvpnclient

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/mysteriumnetwork/node/services/wireguard/connection/dns"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/mysteriumnetwork/node/utils/actionstack"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

type client struct {
	iface      string
	wgClient   *wgctrl.Client
	dnsManager dns.Manager
}

// New creates new wireguard dvpn client.
func New() (*client, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	return &client{
		wgClient:   wgClient,
		dnsManager: dns.NewManager(),
	}, nil
}

func (c *client) ConfigureDevice(config wgcfg.DeviceConfig) error {
	rollback := actionstack.NewActionStack()

	if err := c.up(config.IfaceName); err != nil {
		return err
	}
	rollback.Push(func() {
		_ = c.DestroyDevice(config.IfaceName)
	})

	err := c.configureDevice(config)
	if err != nil {
		rollback.Run()
		return err
	}

	if config.Peer.Endpoint != nil {
		gw := config.Subnet.IP.To4()
		if gw == nil {
			return fmt.Errorf("Subnet %s is not an IPv4 address", config.Subnet.String())
		}

		gw[3]--
		if err := cmdutil.SudoExec("ip", "route", "add", "default", "via", gw.String(), "dev", config.IfaceName, "table", config.IfaceName); err != nil {
			if !strings.Contains(err.Error(), "File exists") {
				// Ignore error if the route already exist.
				return err
			}
		}
	}

	return nil
}

func (c *client) ReConfigureDevice(config wgcfg.DeviceConfig) error {
	return c.configureDevice(config)
}

func (c *client) configureDevice(config wgcfg.DeviceConfig) error {
	if err := cmdutil.SudoExec("ip", "address", "replace", "dev", config.IfaceName, config.Subnet.String()); err != nil {
		return err
	}

	peer, err := peerConfig(config.Peer)
	if err != nil {
		return err
	}

	privateKey, err := stringToKey(config.PrivateKey)
	if err != nil {
		return err
	}

	c.iface = config.IfaceName
	deviceConfig := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &config.ListenPort,
		Peers:        []wgtypes.PeerConfig{peer},
		ReplacePeers: true,
	}

	if err := c.wgClient.ConfigureDevice(c.iface, deviceConfig); err != nil {
		return fmt.Errorf("could not configure kernel space device: %w", err)
	}

	if err := c.dnsManager.Set(dns.Config{
		ScriptDir: config.DNSScriptDir,
		IfaceName: config.IfaceName,
		DNS:       config.DNS,
	}); err != nil {
		return fmt.Errorf("could not set DNS: %w", err)
	}

	return nil
}

func peerConfig(peer wgcfg.Peer) (wgtypes.PeerConfig, error) {
	endpoint := peer.Endpoint
	publicKey, err := stringToKey(peer.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("could not convert string key to wgtypes.Key: %w", err)
	}

	// Apply keep alive interval
	var keepAliveInterval *time.Duration
	if peer.KeepAlivePeriodSeconds > 0 {
		interval := time.Duration(peer.KeepAlivePeriodSeconds) * time.Second
		keepAliveInterval = &interval
	}

	// Apply allowed IPs network
	var allowedIPs []net.IPNet
	for _, ip := range peer.AllowedIPs {
		_, network, err := net.ParseCIDR(ip)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("could not parse IP %q: %v", ip, err)
		}
		allowedIPs = append(allowedIPs, *network)
	}

	return wgtypes.PeerConfig{
		Endpoint:                    endpoint,
		PublicKey:                   publicKey,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: keepAliveInterval,
	}, nil
}

func (c *client) PeerStats(string) (wgcfg.Stats, error) {
	d, err := c.wgClient.Device(c.iface)
	if err != nil {
		return wgcfg.Stats{}, err
	}

	if len(d.Peers) != 1 {
		return wgcfg.Stats{}, errors.New("kernelspace: exactly 1 peer expected")
	}

	return wgcfg.Stats{
		BytesReceived: uint64(d.Peers[0].ReceiveBytes),
		BytesSent:     uint64(d.Peers[0].TransmitBytes),
		LastHandshake: d.Peers[0].LastHandshakeTime,
	}, nil
}

func (c *client) DestroyDevice(name string) error {
	return cmdutil.SudoExec("ip", "link", "del", "dev", name)
}

func (c *client) up(iface string) error {
	rollback := actionstack.NewActionStack()
	if d, err := c.wgClient.Device(iface); err != nil || d.Name != iface {
		if err := cmdutil.SudoExec("ip", "link", "add", "dev", iface, "type", "wireguard"); err != nil {
			return err
		}
	}
	rollback.Push(func() {
		_ = c.DestroyDevice(iface)
	})

	if err := cmdutil.SudoExec("ip", "link", "set", "dev", iface, "up"); err != nil {
		rollback.Run()
		return err
	}

	return nil
}

func (c *client) Close() (err error) {
	errs := utils.ErrorCollection{}
	if err := c.DestroyDevice(c.iface); err != nil {
		errs.Add(err)
	}
	if err := c.wgClient.Close(); err != nil {
		errs.Add(err)
	}
	if err := c.dnsManager.Clean(); err != nil {
		errs.Add(err)
	}
	if err := errs.Error(); err != nil {
		return fmt.Errorf("could not close client: %w", err)
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
