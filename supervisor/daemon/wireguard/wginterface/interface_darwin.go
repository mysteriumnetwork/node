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
	"fmt"
	"net"
	"os"
	"path"
	"strconv"

	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

func createTunnel(requestedInterfaceName string, _ []string) (tunnel tun.Device, interfaceName string, err error) {
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

func newUAPIListener(interfaceName string) (listener net.Listener, err error) {
	log.Info().Msg("Setting interface configuration")
	fileUAPI, err := ipc.UAPIOpen(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("UAPI listen error: %w", err)
	}
	uapi, err := ipc.UAPIListen(interfaceName, fileUAPI)
	if err != nil {
		return nil, fmt.Errorf("could not listen for UAPI wg configuration: %w", err)
	}
	return uapi, nil
}

// applySocketPermissions changes ownership of the WireGuard socket to the given user.
func applySocketPermissions(interfaceName string, uid string) error {
	numUid, err := strconv.Atoi(uid)
	if err != nil {
		return fmt.Errorf("failed to parse uid %s: %w", uid, err)
	}
	socketPath := path.Join("/var/run/wireguard", fmt.Sprintf("%s.sock", interfaceName))
	err = os.Chown(socketPath, numUid, -1)
	if err != nil {
		return fmt.Errorf("failed to chown wireguard socket to uid %s: %w", uid, err)
	}
	return nil
}

func disableFirewall() {}
