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
	"runtime"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/diagclient"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/dvpnclient"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/kernelspace"
	netstack_provider "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack-provider"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/proxyclient"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/remoteclient"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
)

// WgClient represents WireGuard client.
type WgClient interface {
	ConfigureDevice(config wgcfg.DeviceConfig) error
	ReConfigureDevice(config wgcfg.DeviceConfig) error
	DestroyDevice(name string) error
	PeerStats(iface string) (wgcfg.Stats, error)
	Close() error
}

// WgClientDiag is a specialised WgClient interface for provider check
type WgClientDiag interface {
	Diag() bool
}

// WgClientFactory represents WireGuard client factory.
type WgClientFactory struct {
	once                         sync.Once
	isKernelSpaceSupportedResult bool
}

// NewWGClientFactory returns a new client factory.
func NewWGClientFactory() *WgClientFactory {
	return &WgClientFactory{}
}

// NewWGClient returns a new wireguard client.
func (wcf *WgClientFactory) NewWGClient() (WgClient, error) {

	if config.GetBool(config.FlagProvCheckerMode) {
		return diagclient.New()
	}
	if config.GetBool(config.FlagDVPNMode) {
		return dvpnclient.New()
	}

	if config.GetBool(config.FlagProxyMode) {
		return proxyclient.New()
	}

	if config.GetBool(config.FlagUserspace) {
		return netstack_provider.New()
	}

	if config.GetBool(config.FlagUserMode) {
		return remoteclient.New()
	}

	wcf.once.Do(func() {
		wcf.isKernelSpaceSupportedResult = wcf.isKernelSpaceSupported()
	})

	if wcf.isKernelSpaceSupportedResult {
		return kernelspace.NewWireguardClient()
	}

	log.Info().Msg("Wireguard kernel space is not supported. Switching to user space implementation.")

	return userspace.NewWireguardClient()
}

func (wcf *WgClientFactory) isKernelSpaceSupported() bool {
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
