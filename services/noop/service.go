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
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

const logPrefix = "[service-noop] "

// NewManager creates new instance of Noop service
func NewManager() *Manager {
	return &Manager{}
}

// Manager represents entrypoint for Noop service
type Manager struct{}

// Start starts service - does not block
func (manager *Manager) Start(providerID identity.Identity) (
	proposal dto_discovery.ServiceProposal,
	sessionConfigProvider session.ConfigProvider,
	err error,
) {
	proposal = dto_discovery.ServiceProposal{
		ServiceType: "noop",
		ServiceDefinition: ServiceDefinition{
			Location: dto_discovery.Location{Country: ""},
		},
		PaymentMethodType: PaymentMethodNoop,
		PaymentMethod: PaymentNoop{
			Price: money.NewMoney(0, money.CURRENCY_MYST),
		},
	}
	sessionConfigProvider = func() (session.ServiceConfiguration, error) {
		return nil, nil
	}
	log.Info(logPrefix, "Openvpn service started successfully")
	return
}

// Wait blocks until service is stopped
func (manager *Manager) Wait() error {
	return nil
}

// Stop stops service
func (manager *Manager) Stop() error {
	return nil
}
