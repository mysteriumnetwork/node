// +build linux,!android

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

package endpoint

import (
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/kernelspace"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/utils"
)

// NewConnectionEndpoint creates new wireguard connection endpoint.
func NewConnectionEndpoint(
	ipResolver ip.Resolver,
	resourceAllocator *resources.Allocator,
	mapPort func(port int) (releasePortMapping func()),
	connectDelay int) (wg.ConnectionEndpoint, error) {

	wgClient, err := getWGClient()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		wgClient:           wgClient,
		ipResolver:         ipResolver,
		resourceAllocator:  resourceAllocator,
		releasePortMapping: func() {},
		mapPort:            mapPort,
		connectDelay:       connectDelay,
	}, nil
}

func getWGClient() (wgClient wgClient, err error) {
	if isKernelSpaceSupported() {
		return kernelspace.NewWireguardClient()
	}

	log.Info("Wireguard kernel space is not supported. Switching to user space implementation.")
	return userspace.NewWireguardClient()
}

func isKernelSpaceSupported() bool {
	err := utils.SudoExec("ip", "link", "add", "iswgsupported", "type", "wireguard")
	if err != nil {
		log.Debug(logPrefix, "failed to create wireguard network interface: ", err)
	}

	_ = utils.SudoExec("ip", "link", "del", "iswgsupported")
	return err == nil
}
