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
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-wireguard] "

// NewManager creates new instance of Wireguard service
func NewManager(locationResolver location.Resolver, ipResolver ip.Resolver, connectionEndpoint wg.ConnectionEndpoint) *Manager {
	return &Manager{
		locationResolver:   locationResolver,
		ipResolver:         ipResolver,
		connectionEndpoint: connectionEndpoint,
		natService:         nat.NewService(),
	}
}

// Manager represents entrypoint for Wireguard service.
type Manager struct {
	locationResolver   location.Resolver
	ipResolver         ip.Resolver
	connectionEndpoint wg.ConnectionEndpoint
	wg                 sync.WaitGroup
	natService         nat.NATService
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (market.ServiceProposal, session.ConfigProvider, error) {
	if err := manager.connectionEndpoint.Start(nil); err != nil {
		return market.ServiceProposal{}, nil, err
	}
	config, err := manager.connectionEndpoint.Config()
	if err != nil {
		return market.ServiceProposal{}, nil, err
	}

	outboundIP, err := manager.ipResolver.GetOutboundIP()
	if err != nil {
		return market.ServiceProposal{}, nil, err
	}
	manager.natService.Add(nat.RuleForwarding{
		SourceAddress: config.Consumer.IPAddress.String(),
		TargetIP:      outboundIP,
	})
	if err := manager.natService.Start(); err != nil {
		return market.ServiceProposal{}, nil, err
	}

	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		privateKey, err := endpoint.GeneratePrivateKey()
		if err != nil {
			return wg.ServiceConfig{}, err
		}

		publicKey, err := endpoint.PrivateKeyToPublicKey(privateKey)
		if err != nil {
			return wg.ServiceConfig{}, err
		}

		if err := manager.connectionEndpoint.AddPeer(publicKey, nil); err != nil {
			return wg.ServiceConfig{}, err
		}

		config, err := manager.connectionEndpoint.Config()
		if err != nil {
			return wg.ServiceConfig{}, err
		}

		config.Consumer.PrivateKey = privateKey
		return config, nil
	}

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

	return proposal, sessionConfigProvider, nil
}

// Wait blocks until service is stopped.
func (manager *Manager) Wait() error {
	manager.wg.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	manager.wg.Done()
	manager.natService.Stop()
	if err := manager.connectionEndpoint.Stop(); err != nil {
		return err
	}

	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}
