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

	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("provider-id")
)

var _ service.Service = NewManager()

func Test_GetProposal(t *testing.T) {
	country := "LT"
	assert.Exactly(
		t,
		market.ServiceProposal{
			ServiceType: "noop",
			ServiceDefinition: ServiceDefinition{
				Location: market.Location{Country: country},
			},

			PaymentMethodType: "NOOP",
			PaymentMethod: PaymentNoop{
				Price: pingpong.DefaultPaymentInfo.Price,
			},
		},
		GetProposal(location.Location{Country: country}),
	)
}

func Test_Manager_ProvideConfig(t *testing.T) {
	manager := NewManager()
	sessionConfig, err := manager.ProvideConfig(nil)
	assert.NoError(t, err)
	assert.NotNil(t, sessionConfig.TraversalParams)
	assert.Nil(t, sessionConfig.SessionServiceConfig)
	assert.Nil(t, sessionConfig.SessionDestroyCallback)
}

func Test_Manager_Serve_Stop(t *testing.T) {
	manager := NewManager()
	go func() {
		err := manager.Serve(&service.Instance{})
		assert.NoError(t, err)
	}()

	time.Sleep(time.Millisecond * 10)
	err := manager.Stop()
	assert.NoError(t, err)
}
