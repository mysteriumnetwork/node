/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package endpoint

import (
	"net"
	"runtime"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/kernelspace"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
	"github.com/rs/zerolog/log"
)

type wgClient interface {
	ConfigureDevice(config wg.DeviceConfig) error
	ConfigureRoutes(iface string, ip net.IP) error
	DestroyDevice(name string) error
	AddPeer(iface string, peer wg.Peer) error
	RemovePeer(name string, publicKey string) error
	PeerStats() (*wg.Stats, error)
	Close() error
}

func newWGClient() (wgClient, error) {
	if isKernelSpaceSupported() {
		return kernelspace.NewWireguardClient()
	}

	log.Info().Msg("Wireguard kernel space is not supported. Switching to user space implementation.")
	return userspace.NewWireguardClient()
}

func isKernelSpaceSupported() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	err := cmdutil.SudoExec("ip", "link", "add", "iswgsupported", "type", "wireguard")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create wireguard network interface")
	}

	if err := cmdutil.SudoExec("ip", "link", "del", "iswgsupported"); err != nil {
		log.Warn().Err(err).Msg("Failed to delete iswgsupported wireguard network interface")
	}
	return err == nil
}
