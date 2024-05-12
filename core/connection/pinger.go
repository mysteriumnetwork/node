/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

// ProviderChecker is a service for provider testing
type ProviderChecker struct {
	bus eventbus.Publisher
}

// NewProviderChecker is a ProviderChecker constructor
func NewProviderChecker(bus eventbus.Publisher) *ProviderChecker {
	return &ProviderChecker{
		bus: bus,
	}
}

// Diag is used to start provider check
func (p *ProviderChecker) Diag(cm *connectionManager, providerID string) {
	c, ok := cm.activeConnection.(ConnectionDiag)
	res := false
	if ok {
		log.Debug().Msgf("Check provider> %v", providerID)

		res = c.Diag()
		cm.Disconnect()
	}
	ev := quality.DiagEvent{ProviderID: providerID, Result: res}
	p.bus.Publish(quality.AppTopicConnectionDiagRes, ev)
}
