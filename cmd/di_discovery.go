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

package cmd

import (
	"time"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/discovery/apidiscovery"
	"github.com/mysteriumnetwork/node/core/discovery/brokerdiscovery"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/pkg/errors"
)

func (di *Dependencies) bootstrapDiscoveryComponents(options node.OptionsDiscovery) error {
	proposalRepository := discovery.NewRepository()
	discoveryRegistry := discovery.NewRegistry()
	for _, discoveryType := range options.Types {
		switch discoveryType {
		case node.DiscoveryTypeAPI:
			discoveryRegistry.AddRegistry(apidiscovery.NewRegistry(di.MysteriumAPI))
			proposalRepository.Add(apidiscovery.NewRepository(di.MysteriumAPI))
		case node.DiscoveryTypeBroker:
			discoveryRegistry.AddRegistry(brokerdiscovery.NewRegistry(di.BrokerConnection))

			brokerRepository := brokerdiscovery.NewRepository(di.BrokerConnection, di.EventBus, options.PingInterval+time.Second, 1*time.Second)
			if options.FetchEnabled {
				di.DiscoveryWorker = brokerRepository
				if err := di.DiscoveryWorker.Start(); err != nil {
					return errors.Wrap(err, "failed to enable broker discovery")
				}
			}
			proposalRepository.Add(brokerRepository)
		default:
			return errors.Errorf("unknown discovery adapter: %s", discoveryType)
		}
	}

	di.ProposalRepository = proposalRepository
	di.DiscoveryFactory = func() service.Discovery {
		return discovery.NewService(di.IdentityRegistry, discoveryRegistry, options.PingInterval, di.SignerFactory, di.EventBus)
	}
	return nil
}
