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
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-wireguard] "

// ErrAlreadyStarted is the error we return when the start is called multiple times.
var ErrAlreadyStarted = errors.New("Service already started")

// NewManager creates new instance of Wireguard service
func NewManager(locationResolver location.Resolver, ipResolver ip.Resolver) *Manager {
	return &Manager{
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		resources:        resources.Handler{},
	}
}

// Manager represents entrypoint for Wireguard service.
type Manager struct {
	isStarted          bool
	process            sync.WaitGroup
	locationResolver   location.Resolver
	ipResolver         ip.Resolver
	connectionEndpoint ConfigProvider
	resources          resources.Handler
}

// Config represent a Wireguard service provider configuration that will be passed to the consumer for establishing a connection.
type Config struct {
	Provider Provider
	Consumer Consumer
}

// ConfigProvider represents Wireguard network instance, it provide information
// required for establishing connection between service provider and consumer.
type ConfigProvider interface {
	ProviderConfig(ip net.IPNet) (Provider, error)
	ConsumerConfig(ip net.IPNet) (Consumer, error)
	Close() error
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error) {
	publicIP, err := manager.ipResolver.GetPublicIP()
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	endpoint := net.UDPAddr{IP: net.ParseIP(publicIP), Port: manager.resources.AllocatePort()}
	manager.connectionEndpoint, err = newConnectionEndpoint(manager.resources.AllocateInterface(), endpoint)
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	provider, err := manager.connectionEndpoint.ProviderConfig(manager.resources.AllocateIP())
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		consumer, err := manager.connectionEndpoint.ConsumerConfig(manager.resources.AllocateIP())
		if err != nil {
			return Config{}, nil
		}
		return Config{Provider: provider, Consumer: consumer}, nil
	}

	if manager.isStarted {
		return dto_discovery.ServiceProposal{}, sessionConfigProvider, ErrAlreadyStarted
	}

	manager.process.Add(1)
	manager.isStarted = true
	log.Info(logPrefix, "Wireguard service started successfully")

	country, err := manager.locationResolver.ResolveCountry(publicIP)
	if err != nil {
		return dto_discovery.ServiceProposal{}, nil, err
	}

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
	if !manager.isStarted {
		return nil
	}
	manager.process.Wait()
	return nil
}

// Stop stops service.
func (manager *Manager) Stop() error {
	if !manager.isStarted {
		return nil
	}

	if err := manager.connectionEndpoint.Close(); err != nil {
		return err
	}

	manager.process.Done()
	manager.isStarted = false
	log.Info(logPrefix, "Wireguard service stopped")
	return nil
}
