/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/services/socks5"
	"github.com/stretchr/testify/assert"
)

var (
	providerID  = identity.FromAddress("provider-id")
	servicePort = 8080
	publicIP    = "127.0.0.1"
	outboundIP  = "127.0.0.1"
	country     = "LT"
)

func Test_GetProposal(t *testing.T) {
	assert.Exactly(
		t,
		market.ServiceProposal{
			ServiceType: "socks5",
			ServiceDefinition: socks5.ServiceDefinition{
				Location:          market.Location{Country: country},
				LocationOriginate: market.Location{Country: country},
			},
			PaymentMethodType: "PER_TIME",
			PaymentMethod:     dto.DefaultPaymentInfo,
		},
		GetProposal(location.Location{Country: country}),
	)
}

func Test_Manager_Stop(t *testing.T) {
	manager := newManagerStub(servicePort, publicIP, outboundIP)

	go func() {
		err := manager.Serve(providerID)
		assert.NoError(t, err)
	}()

	waitABit()
	err := manager.Stop()
	assert.NoError(t, err)
}

func Test_Manager_ProviderConfig_FailsWhenSessionConfigIsInvalid(t *testing.T) {
	manager := newManagerStub(servicePort, publicIP, outboundIP)

	params, err := manager.ProvideConfig(nil)

	assert.Nil(t, params)
	assert.Error(t, err)
}

// usually time.Sleep call gives a chance for other goroutines to kick in important when testing async code
func waitABit() {
	time.Sleep(10 * time.Millisecond)
}

func newManagerStub(servicePort int, publicIP, outboundIP string) *Manager {
	return &Manager{
		natPort: func(int) (natPortRelease func()) {
			return func() {}
		},

		servicePort: servicePort,
		publicIP:    publicIP,
		outboundIP:  outboundIP,
	}
}
