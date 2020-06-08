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
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/node/services/wireguard/connection/dns"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

func socketPath(interfaceName string) string {
	return path.Join("/var/run/wireguard", fmt.Sprintf("%s.sock", interfaceName))
}

// New creates new WgInterface instance.
func New(cfg wgcfg.DeviceConfig, uid string) (*WgInterface, error) {
	tunnel, interfaceName, err := createTunnel(cfg.IfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device %s: %w", cfg.IfaceName, err)
	}

	logger := device.NewLogger(device.LogLevelDebug, fmt.Sprintf("(%s) ", interfaceName))
	logger.Info.Println("Starting wireguard-go version", device.WireGuardGoVersion)

	wgDevice := device.NewDevice(tunnel, logger)
	logger.Info.Println("Device started")

	log.Info().Msg("Setting interface configuration")
	fileUAPI, err := ipc.UAPIOpen(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("UAPI listen error: %w", err)
	}
	uapi, err := ipc.UAPIListen(interfaceName, fileUAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UAPI socket: %w", err)
	}
	if err := wgDevice.IpcSetOperation(bufio.NewReader(strings.NewReader(cfg.Encode()))); err != nil {
		return nil, fmt.Errorf("could not set device uapi config: %w", err)
	}

	log.Info().Msg("Bringing peers up")
	wgDevice.Up()

	log.Info().Msg("Configuring network")
	dnsManager := dns.NewManager()
	if err := configureNetwork(cfg, dnsManager); err != nil {
		return nil, fmt.Errorf("could not setup network: %w", err)
	}

	numUid, err := strconv.Atoi(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uid %s: %w", uid, err)
	}
	err = os.Chown(socketPath(interfaceName), numUid, -1) // this won't work on windows
	if err != nil {
		return nil, fmt.Errorf("failed to chown wireguard socket to uid %s: %w", uid, err)
	}

	wgInterface := &WgInterface{
		Name:       interfaceName,
		Device:     wgDevice,
		uapi:       uapi,
		dnsManager: dnsManager,
	}
	log.Info().Msg("Listening for UAPI requests")
	go wgInterface.handleUAPI()

	return wgInterface, nil
}

func createTunnel(requestedInterfaceName string) (tunnel tun.Device, interfaceName string, err error) {
	tunnel, err = tun.CreateTUN(requestedInterfaceName, device.DefaultMTU)
	if err == nil {
		interfaceName = requestedInterfaceName
		realInterfaceName, err2 := tunnel.Name()
		if err2 == nil {
			interfaceName = realInterfaceName
		}
	}
	return tunnel, interfaceName, err
}

// handleUAPI listens for WireGuard configuration changes via user space socket.
func (a *WgInterface) handleUAPI() {
	for {
		conn, err := a.uapi.Accept()
		if err != nil {
			log.Err(err).Msg("Failed to close UAPI listener")
			return
		}
		go a.Device.IpcHandle(conn)
	}
}

func configureNetwork(cfg wgcfg.DeviceConfig, dnsManager dns.Manager) error {
	if err := netutil.AssignIP(cfg.IfaceName, cfg.Subnet); err != nil {
		return fmt.Errorf("failed to assign IP address: %w", err)
	}

	if cfg.Peer.Endpoint != nil {
		if err := netutil.ExcludeRoute(cfg.Peer.Endpoint.IP); err != nil {
			return err
		}
		if err := netutil.AddDefaultRoute(cfg.IfaceName); err != nil {
			return err
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

// Down closes device and user space api socket.
func (a *WgInterface) Down() {
	_ = a.uapi.Close()
	a.Device.Close()
	if err := a.dnsManager.Clean(); err != nil {
		log.Err(err).Msg("Could not clean DNS")
	}
}
