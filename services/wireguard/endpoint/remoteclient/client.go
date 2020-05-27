/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package remoteclient

import (
	"fmt"
	"net"
	"sync"
	"time"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
	supervisorclient "github.com/mysteriumnetwork/node/supervisor/client"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type client struct {
	mu    sync.Mutex
	iface string
	wgc   *wgctrl.Client
}

// New create new remote WireGuard client which communicates with supervisor.
func New() (*client, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("Creating remote wg client")
	return &client{
		wgc: wgc,
	}, nil
}

func (c *client) ConfigureDevice(config wg.DeviceConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.iface = config.IfaceName
	if err := createTUN(c.iface, config.Subnet); err != nil {
		return fmt.Errorf("failed to create TUN device %s: %w", c.iface, err)
	}

	key, err := wgtypes.ParseKey(config.PrivateKey)
	if err != nil {
		return fmt.Errorf("could not parse private key: %w", err)
	}
	err = c.wgc.ConfigureDevice(c.iface, wgtypes.Config{
		PrivateKey: &key,
		ListenPort: &config.ListenPort,
	})
	if err != nil {
		return fmt.Errorf("could not configure device: %w", err)
	}
	return nil
}

func (c *client) ConfigureRoutes(iface string, ip net.IP) error {
	if err := excludeRoute(ip); err != nil {
		return err
	}
	return addDefaultRoute(iface)
}

func (c *client) DestroyDevice(iface string) error {
	err := destroyTUN(iface)
	if err != nil {
		return fmt.Errorf("could not destroy TUN device %s: %w", iface, err)
	}
	return nil
}

func (c *client) AddPeer(iface string, peer wg.Peer) error {
	key, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return fmt.Errorf("could not parse private key: %w", err)
	}
	duration := time.Second * time.Duration(peer.KeepAlivePeriodSeconds)
	var allowedIPs []net.IPNet
	for _, s := range peer.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return fmt.Errorf("could not parse IPNet: %s: %w", s, err)
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}
	return c.wgc.ConfigureDevice(iface, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:                   key,
				Endpoint:                    peer.Endpoint,
				PersistentKeepaliveInterval: &duration,
				AllowedIPs:                  allowedIPs,
			},
		},
	})
}

func (c *client) RemovePeer(iface string, publicKey string) error {
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("could not parse private key: %w", err)
	}
	return c.wgc.ConfigureDevice(iface, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				Remove:    true,
				PublicKey: key,
			},
		},
	})
}

func (c *client) PeerStats(iface string) (*wg.Stats, error) {
	d, err := c.wgc.Device(iface)
	if err != nil {
		return nil, err
	}
	if len(d.Peers) != 1 {
		return nil, fmt.Errorf("exactly 1 peer expected, got %d", len(d.Peers))
	}

	p := d.Peers[0]
	return &wg.Stats{
		BytesSent:     uint64(p.TransmitBytes),
		BytesReceived: uint64(p.ReceiveBytes),
		LastHandshake: p.LastHandshakeTime,
	}, nil
}

func (c *client) Close() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	errs := utils.ErrorCollection{}
	if err := c.DestroyDevice(c.iface); err != nil {
		errs.Add(err)
	}
	if err := c.wgc.Close(); err != nil {
		errs.Add(err)
	}
	if err := errs.Error(); err != nil {
		return fmt.Errorf("could not close client: %w", err)
	}
	return nil
}

func excludeRoute(ip net.IP) error {
	_, err := supervisorclient.Command("exclude-route", "-ip", ip.String())
	return err
}

func addDefaultRoute(iface string) error {
	_, err := supervisorclient.Command("default-route", "-iface", iface)
	return err
}
