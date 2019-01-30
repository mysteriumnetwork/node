// +build darwin linux,!android windows

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

package cmd

import (
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
)

// BootstrapServices loads all the components required for running services
func (di *Dependencies) BootstrapServices(nodeOptions node.Options) error {
	di.bootstrapServiceComponents(nodeOptions)

	di.bootstrapServiceOpenvpn(nodeOptions)
	di.bootstrapServiceNoop(nodeOptions)
	di.bootstrapServiceWireguard(nodeOptions)

	return nil
}

func (di *Dependencies) bootstrapServiceWireguard(nodeOptions node.Options) {
	di.ServiceRegistry.Register(
		wireguard.ServiceType,
		func(_ string, serviceOptions service.Options) (service.Service, market.ServiceProposal, error) {
			location, err := di.resolveIPsAndLocation()
			if err != nil {
				return nil, market.ServiceProposal{}, err
			}

			return wireguard_service.NewManager(location.PubIP, location.OutIP, location.Country, di.NATService), wireguard_service.GetProposal(location.Country), nil
		},
	)
}
