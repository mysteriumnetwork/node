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
	locationDatacenter  = market.Location{ASN: 1000, Country: "DE", City: "Berlin", IPType: "datacenter"}
	locationResidential = market.Location{ASN: 124, Country: "LT", City: "Vilnius", IPType: "residential"}

	proposalEmpty              = market.NewProposal("0xbeef", "empty", market.NewProposalOpts{})
	proposalProvider1Streaming = market.NewProposal(provider1, serviceTypeStreaming, market.NewProposalOpts{
		Location:       &locationDatacenter,
		AccessPolicies: []market.AccessPolicy{accessRuleWhitelist},
	})
	proposalProvider1Noop      = market.NewProposal(provider1, serviceTypeNoop, market.NewProposalOpts{})
	proposalProvider2Streaming = market.NewProposal(provider2, serviceTypeStreaming, market.NewProposalOpts{
		Location:       &locationResidential,
		AccessPolicies: []market.AccessPolicy{accessRuleWhitelist, accessRuleBlacklist},
	})
	proposalSupported = market.NewProposal("0xbeef", serviceTypeNoop, market.NewProposalOpts{
		Contacts: []market.Contact{{
			Type:       "phone",
			Definition: "69935951",
		}},
	})
)

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

func Test_ProposalFilter_FiltersByLocationCountry(t *testing.T) {
	filter := &Filter{
		LocationCountry: "DE",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		LocationCountry: "XXX",
	}
	assert.False(t, filter.Matches(proposalProvider1Streaming))
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
		IPType: "datacenter",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		IPType: "residential",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_FiltersByAccessID(t *testing.T) {
	filter := &Filter{
		AccessPolicy: "whitelist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		AccessPolicy: "blacklist",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.True(t, filter.Matches(proposalProvider2Streaming))

	filter = &Filter{
		AccessPolicy:       "whitelist",
		AccessPolicySource: "unknown.txt",
	}
	assert.False(t, filter.Matches(proposalEmpty))
	assert.False(t, filter.Matches(proposalProvider1Streaming))
	assert.False(t, filter.Matches(proposalProvider1Noop))
	assert.False(t, filter.Matches(proposalProvider2Streaming))
}

func Test_ProposalFilter_Filters_Unsupported(t *testing.T) {
	filter := &Filter{ExcludeUnsupported: true}

	market.RegisterServiceType(serviceTypeNoop)
	assert.False(t, filter.Matches(proposalEmpty))
	assert.True(t, filter.Matches(proposalSupported))
}
