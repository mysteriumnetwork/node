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

package noop

import (
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-noop] "

// NewManager creates new instance of Noop service
func NewManager(resolver location.Resolver) *Manager {
	return &Manager{locationResolver: resolver}
}

// Manager represents entrypoint for Noop service
type Manager struct {
	fakeProcess      sync.WaitGroup
	locationResolver location.Resolver
}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (dto_discovery.ServiceProposal, session.ConfigProvider, error) {
	manager.fakeProcess.Add(1)
	log.Info(logPrefix, "Noop service started successfully")

	proposal := dto_discovery.ServiceProposal{
		ServiceType: ServiceType,
		ServiceDefinition: ServiceDefinition{
			Location: dto_discovery.Location{Country: "LT"},
		},
		PaymentMethodType: PaymentMethodNoop,
		PaymentMethod: PaymentNoop{
			Price: money.NewMoney(0, money.CURRENCY_MYST),
		},
	}
	sessionConfigProvider := func() (session.ServiceConfiguration, error) {
		return nil, nil
	}
	return proposal, sessionConfigProvider, nil
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	manager.fakeProcess.Wait()
	return nil
}

// Stop stops service
func (manager *Manager) Stop() error {
	manager.fakeProcess.Done()

	log.Info(logPrefix, "Noop service stopped")
	return nil
}

// GetType returns the service type
func (manager *Manager) GetType() string {
	return ServiceType
}
