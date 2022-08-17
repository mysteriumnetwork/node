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

package netstack_provider

import (
	"bufio"
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"

	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

type client struct {
	mu     sync.Mutex
	Device *device.Device
}

// New create new WireGuard client in full userspace environment using netstack.
func New() (*client, error) {
	log.Debug().Msg("Creating userspace wg client")
	return &client{}, nil
}

func (c *client) ReConfigureDevice(config wgcfg.DeviceConfig) error {
	return c.ConfigureDevice(config)
}

func (c *client) ConfigureDevice(cfg wgcfg.DeviceConfig) error {
	tunnel, _, _, err := CreateNetTUNWithStack([]netip.Addr{netip.MustParseAddr(cfg.Subnet.IP.String())}, cfg.DNSPort, device.DefaultMTU)
	if err != nil {
		return fmt.Errorf("failed to create netstack device %s: %w", cfg.IfaceName, err)
	}

	logger := device.NewLogger(device.LogLevelVerbose, fmt.Sprintf("(%s) ", cfg.IfaceName))
	wgDevice := device.NewDevice(tunnel, conn.NewDefaultBind(), logger)

	log.Info().Msg("Applying interface configuration")
	if err := wgDevice.IpcSetOperation(bufio.NewReader(strings.NewReader(cfg.Encode()))); err != nil {
		wgDevice.Close()
		return fmt.Errorf("could not set device uapi config: %w", err)
	}

	log.Info().Msg("Bringing device up")
	wgDevice.Up()

	c.mu.Lock()
	c.Device = wgDevice
	c.mu.Unlock()

	return nil
}

func (c *client) DestroyDevice(iface string) error {
	return c.Close()
}

func (c *client) PeerStats(iface string) (wgcfg.Stats, error) {
	deviceState, err := userspace.ParseUserspaceDevice(c.Device.IpcGetOperation)
	if err != nil {
		return wgcfg.Stats{}, fmt.Errorf("could not parse device state: %w", err)
	}

	stats, statErr := userspace.ParseDevicePeerStats(deviceState)
	if err != nil {
		err = statErr
		log.Warn().Err(err).Msg("Failed to parse device stats, will try again")
	} else {
		return stats, nil
	}

	return wgcfg.Stats{}, fmt.Errorf("could not parse device state: %w", err)
}

func (c *client) Close() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Device != nil {
		c.Device.Close()
	}

	return nil
}
