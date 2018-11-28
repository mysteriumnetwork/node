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

package wireguard

import (
	"errors"
	"net"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-wireguard] "

// ErrAlreadyStarted is the error we return when the start is called multiple times.
var ErrAlreadyStarted = errors.New("Service already started")

// NewManager creates new instance of Wireguard service
func NewManager(locationResolver location.Resolver, ipResolver ip.Resolver, connectionEndpoint ConnectionEndpoint) *Manager {
	return &Manager{
		locationResolver:   locationResolver,
		ipResolver:         ipResolver,
		connectionEndpoint: connectionEndpoint,
	}
}

// Manager represents entrypoint for Wireguard service.
type Manager struct {
	locationResolver   location.Resolver
	ipResolver         ip.Resolver
	connectionEndpoint ConnectionEndpoint
	wg                 sync.WaitGroup
}

// serviceConfig represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection.
type serviceConfig struct {
	Provider struct {
		IP        net.IPNet
		PublicKey wgtypes.Key
		Endpoint  net.UDPAddr
	}
	Consumer struct {
		IP         net.IPNet
		PrivateKey wgtypes.Key // TODO peer private key should be generated on consumer side
	}
}

// ConnectionEndpoint represents Wireguard network instance, it provide information
// required for establishing connection between service provider and consumer.
type ConnectionEndpoint interface {
	Start() error
	NewConsumer() (configProvider, error)
	Stop() error
}

type configProvider interface {
	Config() (serviceConfig, error)
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error) {
	if err := manager.connectionEndpoint.Start(); err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		consumer, err := manager.connectionEndpoint.NewConsumer()
		if err != nil {
			return serviceConfig{}, err
		}
		return consumer.Config()
	}

	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	country, err := manager.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	manager.wg.Add(1)
	log.Info(logPrefix, "Wireguard service started successfully")

	proposal := dto_discovery.ServiceProposal{
		ServiceType: ServiceType,
		ServiceDefinition: ServiceDefinition{
			Location: dto_discovery.Location{Country: country},
		},
		PaymentMethodType: PaymentMethod,
		PaymentMethod: Payment{
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
	if err := manager.connectionEndpoint.Stop(); err != nil {
		return err
	}

	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}
