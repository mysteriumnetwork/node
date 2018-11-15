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
	"testing"

	"time"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
)

var _ service.Service = NewManager()

func Test_Manager_Start(t *testing.T) {
	manager := &Manager{}
	proposal, sessionConfigProvider, err := manager.Start(providerID)
	assert.NoError(t, err)

	assert.Exactly(
		t,
		dto_discovery.ServiceProposal{
			ServiceType: "noop",
			ServiceDefinition: ServiceDefinition{
				Location: dto_discovery.Location{Country: ""},
			},

			PaymentMethodType: "NOOP",
			PaymentMethod: PaymentNoop{
				Price: money.Money{0, money.Currency("MYST")},
			},
		},
		proposal,
	)

	sessionConfig, err := sessionConfigProvider()
	assert.NoError(t, err)
	assert.Nil(t, sessionConfig)
}

func Test_Manager_Wait(t *testing.T) {
	manager := &Manager{}
	manager.Start(providerID)

	go func() {
		manager.Wait()
		assert.Fail(t, "Wait should be blocking")
	}()

	waitABit()
}

func Test_Manager_Stop(t *testing.T) {
	manager := &Manager{}
	manager.Start(providerID)

	err := manager.Stop()
	assert.NoError(t, err)

	// Wait should not block after stopping
	manager.Wait()
}

// usually time.Sleep call gives a chance for other goroutines to kick in important when testing async code
func waitABit() {
	time.Sleep(10 * time.Millisecond)
}
