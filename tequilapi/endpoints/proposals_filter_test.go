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

package endpoints

import (
	"testing"

	"github.com/mysteriumnetwork/node/market"
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
)

type mockService struct {
	Location market.Location
}

func (service mockService) GetLocation() market.Location {
	return service.Location
}

func Test_ProposalFilter_FiltersAll(t *testing.T) {
	filter := &proposalsFilter{}
	assert.True(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByProviderID(t *testing.T) {
	filter := &proposalsFilter{
		providerID: provider1,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByServiceType(t *testing.T) {
	filter := &proposalsFilter{
		serviceType: serviceTypeNoop,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.True(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &proposalsFilter{
		serviceType: serviceTypeStreaming,
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByLocationType(t *testing.T) {
	filter := &proposalsFilter{
		locationType: "datacenter",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &proposalsFilter{
		locationType: "residential",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByAccessID(t *testing.T) {
	filter := &proposalsFilter{
		accessPolicyID: "whitelist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &proposalsFilter{
		accessPolicyID: "blacklist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &proposalsFilter{
		accessPolicyID:     "whitelist",
		accessPolicySource: "unknown.txt",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))
}
