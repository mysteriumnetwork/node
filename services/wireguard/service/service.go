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

package service

import (
	"sync"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
)

const logPrefix = "[service-wireguard] "

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

// GetProposal returns the proposal for wireguard service
func GetProposal(country string) market.ServiceProposal {
	return market.ServiceProposal{
		ServiceType: wg.ServiceType,
		ServiceDefinition: wg.ServiceDefinition{
			Location:          market.Location{Country: country},
			LocationOriginate: market.Location{Country: country},
		},
		PaymentMethodType: wg.PaymentMethod,
		PaymentMethod: wg.Payment{
			Price: money.NewMoney(0, money.CurrencyMyst),
		},
	}
}
