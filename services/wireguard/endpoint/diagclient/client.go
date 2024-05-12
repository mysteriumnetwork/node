/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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

package diagclient

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"

	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

type client struct {
	mu     sync.Mutex
	Device *device.Device
	tnet   *netstack.Net
}

// New create new WireGuard client testing the provider.
func New() (*client, error) {
	log.Error().Msg("Creating pinger wg client")
	return &client{}, nil
}

func (c *client) ReConfigureDevice(config wgcfg.DeviceConfig) error {
	return c.ConfigureDevice(config)
}

func (c *client) ConfigureDevice(cfg wgcfg.DeviceConfig) error {
	localAddr, err := netip.ParseAddr(cfg.Subnet.IP.String())
	if err != nil {
		return fmt.Errorf("could not parse local addr: %w", err)
	}
	if len(cfg.DNS) == 0 {
		return fmt.Errorf("DNS addr list is empty")
	}
	dnsAddr, err := netip.ParseAddr(cfg.DNS[0])
	if err != nil {
		return fmt.Errorf("could not parse DNS addr: %w", err)
	}
	tunnel, tnet, err := netstack.CreateNetTUN([]netip.Addr{localAddr}, []netip.Addr{dnsAddr}, device.DefaultMTU)
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
	c.tnet = tnet

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
	if statErr != nil {
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

	log.Error().Err(err).Msg("Shutting down pinger ...")

	if c.Device != nil {
		go func() {
			time.Sleep(5 * time.Second)
			c.Device.Close()
		}()
	}
	return nil
}

func (c *client) Diag() bool {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: c.tnet.DialContext,
		},
	}
	resp, err := client.Get("http://1.1.1.1/")
	if err != nil {
		log.Error().Err(err).Msg("Get failed")
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Readall failed")
		return false
	}
	_ = body

	return true
}
