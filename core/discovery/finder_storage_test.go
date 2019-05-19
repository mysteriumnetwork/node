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
	"sync"
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	provider1 = "0x1"
	provider2 = "0x2"

	proposalProvider1Streaming = market.ServiceProposal{
		ServiceType: "streaming",
		ProviderID:  provider1,
	}
	proposalProvider1Noop = market.ServiceProposal{
		ServiceType: "noop",
		ProviderID:  provider1,
	}
	proposalProvider2Streaming = market.ServiceProposal{
		ServiceType: "streaming",
		ProviderID:  provider2,
	}

	proposalsStorage = &fakeStorage{
		proposals: []market.ServiceProposal{
			proposalProvider1Streaming,
			proposalProvider1Noop,
			proposalProvider2Streaming,
		},
	}
)

type fakeStorage struct {
	proposals []market.ServiceProposal
	mutex     sync.Mutex
}

func (storage *fakeStorage) Proposals() []market.ServiceProposal {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	return storage.proposals
}

func Test_Finder_GetProposalExisting(t *testing.T) {
	finder := NewFinderStorage(proposalsStorage.Proposals)

	proposal, err := finder.GetProposal(market.ProposalID{ServiceType: "streaming", ProviderID: provider1})
	assert.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.Exactly(t, proposalProvider1Streaming, *proposal)
}

func Test_Finder_GetProposalUnknown(t *testing.T) {
	finder := NewFinderStorage(proposalsStorage.Proposals)

	proposal, err := finder.GetProposal(market.ProposalID{ServiceType: "unknown", ProviderID: "0x100"})
	assert.NoError(t, err)
	assert.Nil(t, proposal)
}

func Test_Finder_GetProposalsAll(t *testing.T) {
	finder := NewFinderStorage(proposalsStorage.Proposals)

	proposals, err := finder.FindProposals(market.ProposalFilter{})
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
	assert.Exactly(t,
		[]market.ServiceProposal{proposalProvider1Streaming, proposalProvider1Noop, proposalProvider2Streaming},
		proposals,
	)
}
