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

	"github.com/mysteriumnetwork/node/core/discovery/reducer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/stretchr/testify/assert"
)

var (
	proposalProvider1Streaming = market.ServiceProposal{
		ServiceType: "streaming",
		ProviderID:  "0x1",
	}
	proposalProvider1Noop = market.ServiceProposal{
		ServiceType: "noop",
		ProviderID:  "0x1",
	}
	proposalProvider2Streaming = market.ServiceProposal{
		ServiceType: "streaming",
		ProviderID:  "0x2",
	}
)

type filter struct {
	serviceType string
}

func (filter *filter) Matches(proposal market.ServiceProposal) bool {
	return filter.serviceType == "" || proposal.ServiceType == filter.serviceType
}

func (filter *filter) ToAPIQuery() mysterium.ProposalsQuery {
	return mysterium.ProposalsQuery{
		ServiceType: filter.serviceType,
	}
}

func Test_Storage_List(t *testing.T) {
	storage := NewStorage()
	assert.Equal(t, []market.ServiceProposal{}, storage.Proposals())

	storage = mockStorage()
	assert.Equal(
		t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop, proposalProvider2Streaming},
		storage.Proposals(),
	)
}

func Test_Finder_MatchProposals(t *testing.T) {
	storage := mockStorage()

	proposals, err := storage.MatchProposals(reducer.All())
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop, proposalProvider2Streaming},
		proposals,
	)

	proposals, err = storage.MatchProposals(reducer.Equal(reducer.ProviderID, "0x2"))
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider2Streaming},
		proposals,
	)
}

func Test_Finder_FindProposals(t *testing.T) {
	storage := mockStorage()

	proposals, err := storage.FindProposals(&filter{})
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop, proposalProvider2Streaming},
		proposals,
	)

	proposals, err = storage.FindProposals(&filter{"streaming"})
	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider2Streaming},
		proposals,
	)
}

func Test_Storage_HasProposal(t *testing.T) {
	storage := NewStorage()
	assert.False(t, storage.HasProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x1"}))

	storage = mockStorage()
	assert.False(t, storage.HasProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x1"}))
	assert.True(t, storage.HasProposal(market.ProposalID{ServiceType: "streaming", ProviderID: "0x1"}))
}

func Test_Storage_GetProposal(t *testing.T) {
	storage := NewStorage()
	proposal, err := storage.GetProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x1"})
	assert.EqualError(t, err, "proposal does not exist")
	assert.Nil(t, proposal)

	storage = mockStorage()
	proposal, err = storage.GetProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x1"})
	assert.EqualError(t, err, "proposal does not exist")
	assert.Nil(t, proposal)

	proposal, err = storage.GetProposal(market.ProposalID{ServiceType: "streaming", ProviderID: "0x1"})
	assert.NoError(t, err)
	assert.Exactly(t, proposalProvider1Streaming, *proposal)
}

func Test_Storage_Set(t *testing.T) {
	storage := mockStorage()
	storage.Set([]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop}...)
	assert.Equal(
		t,
		[]market.ServiceProposal{
			proposalProvider1Streaming,
			proposalProvider1Noop,
		},
		storage.proposals,
	)
}

func Test_Storage_AddProposal(t *testing.T) {
	storage := NewStorage()
	storage.AddProposal(proposalProvider1Streaming)
	assert.Equal(
		t,
		[]market.ServiceProposal{
			proposalProvider1Streaming,
		},
		storage.proposals,
	)

	storage.AddProposal(proposalProvider1Streaming)
	assert.Equal(
		t,
		[]market.ServiceProposal{
			proposalProvider1Streaming,
		},
		storage.proposals,
	)

	storage.AddProposal(proposalProvider1Noop)
	assert.Equal(
		t,
		[]market.ServiceProposal{
			proposalProvider1Streaming,
			proposalProvider1Noop,
		},
		storage.proposals,
	)
}

func Test_Storage_RemoveProposal(t *testing.T) {
	storage := NewStorage()
	storage.RemoveProposal(proposalProvider1Streaming)
	assert.Equal(
		t,
		[]market.ServiceProposal{},
		storage.proposals,
	)

	storage = mockStorage()
	storage.RemoveProposal(proposalProvider1Streaming)
	assert.Equal(
		t,
		[]market.ServiceProposal{
			proposalProvider1Noop,
			proposalProvider2Streaming,
		},
		storage.proposals,
	)
}

func mockStorage() *ProposalStorage {
	return &ProposalStorage{
		proposals: []market.ServiceProposal{
			proposalProvider1Streaming,
			proposalProvider1Noop,
			proposalProvider2Streaming,
		},
	}
}
