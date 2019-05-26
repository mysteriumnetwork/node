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
	"github.com/mysteriumnetwork/node/market"
)

var (
	provider1           = "0x1"
	provider2           = "0x2"
	accessRuleWhitelist = market.AccessPolicy{
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
		ServiceType:       "streaming",
		ProviderID:        provider1,
		ServiceDefinition: mockService{Location: locationDatacenter},
		AccessPolicies:    &[]market.AccessPolicy{accessRuleWhitelist},
	}
	proposalProvider1Noop = market.ServiceProposal{
		ServiceType:       "noop",
		ProviderID:        provider1,
		ServiceDefinition: mockService{},
	}
	proposalProvider2Streaming = market.ServiceProposal{
		ServiceType:       "streaming",
		ProviderID:        provider2,
		ServiceDefinition: mockService{Location: locationResidential},
		AccessPolicies:    &[]market.AccessPolicy{accessRuleWhitelist, accessRuleBlacklist},
	}
)

type mockService struct {
	Location market.Location
}

func (service mockService) GetLocation() market.Location {
	return service.Location
}
