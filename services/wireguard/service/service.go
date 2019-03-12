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

package service

import (
	"encoding/json"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/nat"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
)

const logPrefix = "[service-wireguard] "

// NewManager creates new instance of Wireguard service
func NewManager(locationResolver location.Resolver, ipResolver ip.Resolver, natService nat.NATService) *Manager {
	resourceAllocator := resources.NewAllocator()
	return &Manager{
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		natService:       natService,

		connectionEndpointFactory: func() (wg.ConnectionEndpoint, error) {
			return endpoint.NewConnectionEndpoint(ipResolver, &resourceAllocator)
		},
	}
}

// Manager represents an instance of Wireguard service
type Manager struct {
	locationResolver location.Resolver
	ipResolver       ip.Resolver
	wg               sync.WaitGroup
	natService       nat.NATService

	connectionEndpointFactory func() (wg.ConnectionEndpoint, error)

	mu   sync.Mutex // TODO this is a temporary solution to cleanup oldest used wireguard resources.
	list []*func()  // TODO it should be removed once payment bases session cleanup implemented.
}

// ProvideConfig provides the config for consumer
func (manager *Manager) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	key := &wg.ConsumerConfig{}
	err := json.Unmarshal(publicKey, key)
	if err != nil {
		return nil, nil, err
	}

	manager.cleanOldEndpoints()

	connectionEndpoint, err := manager.connectionEndpointFactory()
	if err != nil {
		return nil, nil, err
	}

	if err := connectionEndpoint.Start(nil); err != nil {
		return nil, nil, err
	}

	if err := connectionEndpoint.AddPeer(key.PublicKey, nil); err != nil {
		return nil, nil, err
	}

	config, err := connectionEndpoint.Config()
	if err != nil {
		return nil, nil, err
	}

	outboundIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return nil, nil, err
	}

	natRule := nat.RuleForwarding{SourceAddress: config.Consumer.IPAddress.String(), TargetIP: outboundIP}
	if err := manager.natService.Add(natRule); err != nil {
		return nil, nil, errors.Wrap(err, "failed to add NAT forwarding rule")
	}

	destroy := func() error {
		if err := manager.natService.Del(natRule); err != nil {
			log.Error(logPrefix, "failed to delete NAT forwarding rule: ", err)
		}
		return connectionEndpoint.Stop()
	}

	return config, func() error { manager.once(func() { destroy() }); return nil }, nil
}

// TODO this is a temporary solution to cleanup oldest used wireguard resources.
// TODO it should be removed once payment bases session cleanup implemented.
func (manager *Manager) once(f func()) func() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	once := sync.Once{}
	cleanOnce := func() {
		once.Do(f)
	}
	manager.list = append(manager.list, &cleanOnce)

	return func() {
		cleanOnce()

		manager.mu.Lock()
		defer manager.mu.Unlock()

		for i := range manager.list {
			if manager.list[i] == &cleanOnce {
				manager.list = append(manager.list[0:i], manager.list[i+1:]...)
				return
			}
		}
	}
}

// TODO this is a temporary solution to cleanup oldest used wireguard resources.
// TODO it should be removed once payment bases session cleanup implemented.
func (manager *Manager) cleanOldEndpoints() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if len(manager.list) >= resources.MaxResources-1 {
		log.Warn(logPrefix, "We have reached a maximum number of interfaces. Cleaning up oldest one.")

		f := *manager.list[0]
		f()
		manager.list = manager.list[1:]
	}
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (market.ServiceProposal, session.ConfigNegotiator, error) {
	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return market.ServiceProposal{}, nil, err
	}

	country, err := manager.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return market.ServiceProposal{}, nil, err
	}

	manager.wg.Add(1)
	log.Info(logPrefix, "Wireguard service started successfully")

	proposal := market.ServiceProposal{
		ServiceType: wg.ServiceType,
		ServiceDefinition: wg.ServiceDefinition{
			Location: market.Location{Country: country},
		},
		PaymentMethodType: wg.PaymentMethod,
		PaymentMethod: wg.Payment{
			Price: money.NewMoney(0, money.CURRENCY_MYST),
		},
	}

	return proposal, manager, nil
}

// Wait blocks until service is stopped.
func (manager *Manager) Wait() error {
	manager.wg.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	manager.wg.Done()

	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}
