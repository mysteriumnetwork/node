/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package discovery

import (
	"testing"
	"time"

	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/money"
	"github.com/mysterium/node/openvpn/discovery/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var (
	providerID      = identity.FromAddress("123456")
	providerContact = dto_discovery.Contact{
		Type: "type1",
	}
	locationLTTelia = dto_discovery.Location{"LT", "Vilnius", "AS8764"}
	protocol        = "tcp"
)

func Test_NewServiceProposalWithLocation(t *testing.T) {
	proposal := NewServiceProposalWithLocation(providerID, providerContact, locationLTTelia, protocol)

	assert.NotNil(t, proposal)
	assert.Equal(t, 1, proposal.ID)
	assert.Equal(t, "service-proposal/v1", proposal.Format)
	assert.Equal(t, "openvpn", proposal.ServiceType)
	assert.Equal(
		t,
		dto.ServiceDefinition{
			Location:          locationLTTelia,
			LocationOriginate: locationLTTelia,
			SessionBandwidth:  83886080,
			Protocol:          "tcp",
		},
		proposal.ServiceDefinition,
	)
	assert.Equal(t, dto.PaymentMethodPerTime, proposal.PaymentMethodType)
	assert.Equal(
		t,
		dto.PaymentPerTime{
			Price:    money.Money{12500000, money.Currency("MYST")},
			Duration: 60 * time.Minute,
		},
		proposal.PaymentMethod,
	)
	assert.Equal(t, providerID.Address, proposal.ProviderID)
	assert.Equal(t, []dto_discovery.Contact{providerContact}, proposal.ProviderContacts)
}
