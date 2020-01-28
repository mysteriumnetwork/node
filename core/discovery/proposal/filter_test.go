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

package proposal

import (
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/stretchr/testify/assert"
)

var (
	provider1            = "0x1"
	provider2            = "0x2"
	serviceTypeStreaming = "streaming"
	serviceTypeNoop      = "noop"
	accessRuleWhitelist  = market.AccessPolicy{
		ID:     "whitelist",
		Source: "whitelist.txt",
	}
	accessRuleBlacklist = market.AccessPolicy{
		ID:     "blacklist",
		Source: "blacklist.txt",
	}
	locationDatacenter  = market.Location{ASN: 1000, Country: "DE", City: "Berlin", NodeType: "datacenter"}
	locationResidential = market.Location{ASN: 124, Country: "LT", City: "Vilnius", NodeType: "residential"}

	proposalEmpty              = market.ServiceProposal{}
	proposalProvider1Streaming = market.ServiceProposal{
		ProviderID:        provider1,
		ServiceType:       serviceTypeStreaming,
		ServiceDefinition: mockService{Location: locationDatacenter},
		AccessPolicies:    &[]market.AccessPolicy{accessRuleWhitelist},
	}
	proposalProvider1Noop = market.ServiceProposal{
		ProviderID:        provider1,
		ServiceType:       serviceTypeNoop,
		ServiceDefinition: mockService{},
	}
	proposalProvider2Streaming = market.ServiceProposal{
		ProviderID:        provider2,
		ServiceType:       serviceTypeStreaming,
		ServiceDefinition: mockService{Location: locationResidential},
		AccessPolicies:    &[]market.AccessPolicy{accessRuleWhitelist, accessRuleBlacklist},
	}
	proposalExpensive = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(9999999999999, money.CurrencyMyst),
		},
	}
	proposalCheap = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(0, money.CurrencyMyst),
		},
	}
	proposalExact = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(1000000, money.CurrencyMyst),
		},
	}
)

type mockService struct {
	Location market.Location
}

func (service mockService) GetLocation() market.Location {
	return service.Location
}

func Test_ProposalFilter_FiltersAll(t *testing.T) {
	filter := &Filter{}
	assert.True(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByProviderID(t *testing.T) {
	filter := &Filter{
		ProviderID: provider1,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByServiceType(t *testing.T) {
	filter := &Filter{
		ServiceType: serviceTypeNoop,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		ServiceType: serviceTypeStreaming,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByLocationType(t *testing.T) {
	filter := &Filter{
		LocationType: "datacenter",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		LocationType: "residential",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByAccessID(t *testing.T) {
	filter := &Filter{
		AccessPolicyID: "whitelist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		AccessPolicyID: "blacklist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		AccessPolicyID:     "whitelist",
		AccessPolicySource: "unknown.txt",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_Filters_ByBounds(t *testing.T) {
	var upper uint64 = 1000000
	var lower uint64 = 100
	filter := &Filter{
		UpperPriceBound: &upper,
		LowerPriceBound: &lower,
	}

	assert.True(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalExpensive))
	assert.False(t, filter.Matches(proposalCheap))
	assert.True(t, filter.Matches(proposalExact))

	lower = 0
	filter = &Filter{
		UpperPriceBound: &upper,
		LowerPriceBound: &lower,
	}

	assert.True(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalExpensive))
	assert.True(t, filter.Matches(proposalCheap))
	assert.True(t, filter.Matches(proposalExact))
}

type mockPaymentMethod struct {
	rate        market.PaymentRate
	paymentType string
	price       money.Money
}

func (mpm *mockPaymentMethod) GetPrice() money.Money {
	return mpm.price
}

func (mpm *mockPaymentMethod) GetType() string {
	return mpm.paymentType
}

func (mpm *mockPaymentMethod) GetRate() market.PaymentRate {
	return mpm.rate
}
