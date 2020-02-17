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

package reducer

import (
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
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
	proposalTimeExpensive = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(9999999999999, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerTime: time.Minute,
			},
		},
	}
	proposalTimeCheap = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(0, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerTime: time.Minute,
			},
		},
	}
	proposalTimeExact = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(1000000, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerTime: time.Minute,
			},
		},
	}
	proposalTimeExactSeconds = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(1000000/60, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerTime: time.Second,
			},
		},
	}
	proposalTimeExpensiveSeconds = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(17000, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerTime: time.Second,
			},
		},
	}
	proposalBytesExpensive = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(7000001, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerByte: bytesInGibibyte,
			},
		},
	}
	proposalBytesCheap = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(0, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerByte: bytesInGibibyte,
			},
		},
	}
	proposalBytesExact = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(7000000, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerByte: bytesInGibibyte,
			},
		},
	}
	proposalBytesExactInParts = market.ServiceProposal{
		PaymentMethod: &mockPaymentMethod{
			price: money.NewMoney(50000, money.CurrencyMyst),
			rate: market.PaymentRate{
				PerByte: 7669584,
			},
		},
	}
)

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

type mockService struct {
	Location market.Location
}

func (service mockService) GetLocation() market.Location {
	return service.Location
}

func conditionAlwaysMatch(_ market.ServiceProposal) bool {
	return true
}

func conditionNeverMatch(_ market.ServiceProposal) bool {
	return false
}

func conditionIsProvider1(provider market.ServiceProposal) bool {
	return provider.ProviderID == provider1
}

func conditionIsStreaming(provider market.ServiceProposal) bool {
	return provider.ServiceType == "streaming"
}

func fieldID(proposal market.ServiceProposal) interface{} {
	return proposal.ID
}

func fieldProviderID(proposal market.ServiceProposal) interface{} {
	return proposal.ProviderID
}
