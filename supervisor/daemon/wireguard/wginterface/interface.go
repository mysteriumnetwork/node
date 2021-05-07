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

package wginterface

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"

	"github.com/mysteriumnetwork/node/services/wireguard/connection/dns"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
)

// WgInterface represents WireGuard tunnel with underlying device.
type WgInterface struct {
	Name       string
	Device     *device.Device
	uapi       net.Listener
	dnsManager dns.Manager
}

// New creates new WgInterface instance.
func New(cfg wgcfg.DeviceConfig, uid string) (*WgInterface, error) {
	tunnel, interfaceName, err := createTunnel(cfg.IfaceName, cfg.DNS)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device %s: %w", cfg.IfaceName, err)
	}

	logger := newLogger(device.LogLevelDebug, fmt.Sprintf("(%s) ", interfaceName))
	logger.Info.Println("Starting wireguard-go version", device.WireGuardGoVersion)

	logger.Info.Println("Starting device")
	wgDevice := device.NewDevice(tunnel, logger)

	log.Info().Msg("Creating UAPI listener")
	uapi, err := newUAPIListener(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UAPI socket: %w", err)
	}

	log.Info().Msg("Applying interface configuration")
	if err := wgDevice.IpcSetOperation(bufio.NewReader(strings.NewReader(cfg.Encode()))); err != nil {
		down(uapi, wgDevice, nil)
		return nil, fmt.Errorf("could not set device uapi config: %w", err)
	}

	log.Info().Msg("Bringing device up")
	wgDevice.Up()

	log.Info().Msg("Configuring network")
	dnsManager := dns.NewManager()
	if err := configureNetwork(cfg, dnsManager); err != nil {
		down(uapi, wgDevice, dnsManager)
		return nil, fmt.Errorf("could not setup network: %w", err)
	}

	if err := applySocketPermissions(interfaceName, uid); err != nil {
		down(uapi, wgDevice, dnsManager)
		return nil, fmt.Errorf("could not apply socket permissions: %w", err)
	}

	wgInterface := &WgInterface{
		Name:       interfaceName,
		Device:     wgDevice,
		uapi:       uapi,
		dnsManager: dnsManager,
	}
	log.Info().Msg("Accepting UAPI requests")
	go wgInterface.accept()

	return wgInterface, nil
}

// Accept listens for WireGuard configuration changes via user space socket.
func (a *WgInterface) accept() {
	for {
		conn, err := a.uapi.Accept()
		if err != nil {
			log.Err(err).Msg("Failed to close UAPI listener")
			return
		}
		go a.Device.IpcHandle(conn)
	}
}

func down(uapi net.Listener, d *device.Device, dnsManager dns.Manager) {
	if uapi != nil {
		if err := uapi.Close(); err != nil {
			log.Warn().Err(err).Msg("Could not close uapi socket")
		}
	}
	if d != nil {
		d.Close()
	}

	disableFirewall()

	if dnsManager != nil {
		if err := dnsManager.Clean(); err != nil {
			log.Err(err).Msg("Could not clean DNS")
		}
	}
}

// Down closes device and user space api socket.
func (a *WgInterface) Down() {
	down(a.uapi, a.Device, a.dnsManager)
}

func configureNetwork(cfg wgcfg.DeviceConfig, dnsManager dns.Manager) error {
	if err := netutil.AssignIP(cfg.IfaceName, cfg.Subnet); err != nil {
		return fmt.Errorf("failed to assign IP address: %w", err)
	}

	if cfg.Peer.Endpoint != nil {
		if err := netutil.ExcludeRoute(cfg.Peer.Endpoint.IP); err != nil {
			return fmt.Errorf("could not exclude route %s: %w", cfg.Peer.Endpoint.IP.String(), err)
		}
		if err := netutil.AddDefaultRoute(cfg.IfaceName); err != nil {
			return fmt.Errorf("could not add default route for %s: %w", cfg.IfaceName, err)
		}
	}

	if err := dnsManager.Set(dns.Config{
		ScriptDir: cfg.DNSScriptDir,
		IfaceName: cfg.IfaceName,
		DNS:       cfg.DNS,
	}); err != nil {
		return fmt.Errorf("could not set DNS: %w", err)
	}

	return nil
}
