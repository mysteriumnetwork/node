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
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/discovery/apidiscovery"
	"github.com/mysteriumnetwork/node/core/discovery/brokerdiscovery"
	"github.com/mysteriumnetwork/node/core/discovery/dhtdiscovery"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/pkg/errors"
)

func (di *Dependencies) bootstrapDiscoveryComponents(options node.OptionsDiscovery) error {
	di.FilterPresetStorage = proposal.NewFilterPresetStorage(di.Storage)
	proposalRepository := discovery.NewRepository()
	proposalRegistry := discovery.NewRegistry()
	discoveryWorker := discovery.NewWorker()

	for _, discoveryType := range options.Types {
		switch discoveryType {
		case node.DiscoveryTypeAPI:
			// Broker is the way to announce node presence currently, so enabled by default no matter the users preferences.
			proposalRegistry.AddRegistry(brokerdiscovery.NewRegistry(di.BrokerConnection))
			proposalRepository.Add(apidiscovery.NewRepository(di.MysteriumAPI))

		case node.DiscoveryTypeBroker:
			storage := brokerdiscovery.NewStorage(di.EventBus)
			brokerRepository := brokerdiscovery.NewRepository(di.BrokerConnection, storage, options.PingInterval+time.Second, 1*time.Second)
			if options.FetchEnabled {
				discoveryWorker.AddWorker(brokerRepository)
			}

			proposalRegistry.AddRegistry(brokerdiscovery.NewRegistry(di.BrokerConnection))
			proposalRepository.Add(brokerRepository)

		case node.DiscoveryTypeDHT:
			dhtNode, err := dhtdiscovery.NewNode(
				fmt.Sprintf("/ip4/%s/%s/%d", options.DHT.Address, options.DHT.Protocol, options.DHT.Port),
				options.DHT.BootstrapPeers,
			)
			if err != nil {
				return errors.Wrap(err, "failed to configure DHT node")
			}
			discoveryWorker.AddWorker(dhtNode)

			proposalRegistry.AddRegistry(dhtdiscovery.NewRegistry())
			proposalRepository.Add(dhtdiscovery.NewRepository())

		default:
			return errors.Errorf("unknown discovery adapter: %s", discoveryType)
		}
	}

	di.DiscoveryWorker = discoveryWorker
	if err := di.DiscoveryWorker.Start(); err != nil {
		return errors.Wrap(err, "failed to start discovery")
	}

	di.ProposalRepository = discovery.NewPricedServiceProposalRepository(proposalRepository, di.PricingHelper, di.FilterPresetStorage)
	di.DiscoveryFactory = func() service.Discovery {
		return discovery.NewService(di.IdentityRegistry, proposalRegistry, options.PingInterval, di.SignerFactory, di.EventBus)
	}
	return nil
}
