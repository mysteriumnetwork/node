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

package discovery

import (
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
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
	proposalsStorage = &ProposalStorage{
		proposals: []market.ServiceProposal{
			proposalProvider1Streaming,
			proposalProvider1Noop,
			proposalProvider2Streaming,
		},
	}
)

type mockService struct {
	Location market.Location
}

func (service mockService) GetLocation() market.Location {
	return service.Location
}

func Test_Finder_GetProposalExisting(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposal, err := finder.GetProposal(market.ProposalID{ServiceType: "streaming", ProviderID: provider1})
	assert.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.Exactly(t, proposalProvider1Streaming, *proposal)
}

func Test_Finder_GetProposalUnknown(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposal, err := finder.GetProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x100"})
	assert.NoError(t, err)
	assert.Nil(t, proposal)
}

func Test_Finder_GetProposalsAll(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposals, err := finder.FindProposals(&market.ProposalFilter{})
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop, proposalProvider2Streaming},
		proposals,
	)
}

func Test_Finder_ProposalsByProviderID(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposals, err := finder.FindProposals(&market.ProposalFilter{
		ProviderID: provider1,
	})

	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Exactly(
		t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop},
		proposals,
	)
}

func Test_Finder_GetProposalsByServiceType(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposals, err := finder.FindProposals(&market.ProposalFilter{
		ServiceType: "noop",
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Exactly(
		t,
		[]market.ServiceProposal{proposalProvider1Noop},
		proposals,
	)

	proposals, err = finder.FindProposals(&market.ProposalFilter{
		ServiceType: "streaming",
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Exactly(
		t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider2Streaming},
		proposals,
	)
}

func Test_Finder_GetProposalsByNodeType(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposals, err := finder.FindProposals(&market.ProposalFilter{
		Location: market.LocationFilter{NodeType: "datacenter"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Exactly(t, []market.ServiceProposal{proposalProvider1Streaming}, proposals)

	proposals, err = finder.FindProposals(&market.ProposalFilter{
		Location: market.LocationFilter{NodeType: "residential"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Exactly(t, []market.ServiceProposal{proposalProvider2Streaming}, proposals)
}

func Test_Finder_GetProposalsByAccessID(t *testing.T) {
	finder := NewFinder(proposalsStorage)

	proposals, err := finder.FindProposals(&market.ProposalFilter{
		AccessPolicy: market.AccessPolicyFilter{ID: "whitelist"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Exactly(
		t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider2Streaming},
		proposals,
	)

	proposals, err = finder.FindProposals(&market.ProposalFilter{
		AccessPolicy: market.AccessPolicyFilter{ID: "blacklist"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Exactly(
		t,
		[]market.ServiceProposal{proposalProvider2Streaming},
		proposals,
	)

	proposals, err = finder.FindProposals(&market.ProposalFilter{
		AccessPolicy: market.AccessPolicyFilter{ID: "whitelist", Source: "unknown.txt"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 0)

	proposals, err = finder.FindProposals(&market.ProposalFilter{
		AccessPolicy: market.AccessPolicyFilter{ID: "unknown"},
	})
	assert.NoError(t, err)
	assert.Len(t, proposals, 0)
}
