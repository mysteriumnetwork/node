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
	"errors"
	"sync"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

// ProposalFinder continuously tracks service proposals from discovery service to storage
type ProposalFinder interface {
	Start() error
	Stop()
}

// ProposalReducer proposal match function
type ProposalReducer func(proposal market.ServiceProposal) bool

// ProposalFilter defines interface with proposal match function
type ProposalFilter interface {
	Matches(proposal market.ServiceProposal) bool
	ToAPIQuery() mysterium.ProposalsQuery
}

// NewStorage creates new instance of ProposalStorage
func NewStorage() *ProposalStorage {
	return &ProposalStorage{
		proposals: make([]market.ServiceProposal, 0),
	}
}

// ProposalStorage represents table of currently active proposals in Mysterium Discovery
type ProposalStorage struct {
	proposals []market.ServiceProposal
	mutex     sync.RWMutex
}

// Proposals returns list of proposals in storage
func (storage *ProposalStorage) Proposals() []market.ServiceProposal {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	return storage.proposals
}

// MatchProposals fetches currently active service proposals from discovery by match function
func (storage *ProposalStorage) MatchProposals(match ProposalReducer) ([]market.ServiceProposal, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	proposals := make([]market.ServiceProposal, 0)
	for _, proposal := range storage.proposals {
		if match(proposal) {
			proposals = append(proposals, proposal)
		}
	}
	return proposals, nil
}

// FindProposals fetches currently active service proposals from discovery by given filter
func (storage *ProposalStorage) FindProposals(filter ProposalFilter) ([]market.ServiceProposal, error) {
	return storage.MatchProposals(filter.Matches)
}

// Set puts given list proposals to storage
func (storage *ProposalStorage) Set(proposals []market.ServiceProposal) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.proposals = proposals
}

// HasProposal checks if proposal exists in storage
func (storage *ProposalStorage) HasProposal(id market.ProposalID) bool {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	_, exist := storage.getProposalIndex(id)
	return exist
}

// GetProposal returns proposal from storage
func (storage *ProposalStorage) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	index, exist := storage.getProposalIndex(id)
	if !exist {
		return nil, errors.New("proposal does not exist")
	}
	return &storage.proposals[index], nil
}

// AddProposal appends given proposal to storage
func (storage *ProposalStorage) AddProposal(proposal market.ServiceProposal) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	if _, exist := storage.getProposalIndex(proposal.UniqueID()); exist {
		return
	}
	storage.proposals = append(storage.proposals, proposal)
}

// RemoveProposal take out given proposal from storage
func (storage *ProposalStorage) RemoveProposal(id market.ProposalID) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	if index, exist := storage.getProposalIndex(id); exist {
		storage.proposals = append(storage.proposals[:index], storage.proposals[index+1:]...)
	}
}

func (storage *ProposalStorage) getProposalIndex(id market.ProposalID) (int, bool) {
	for index, proposal := range storage.proposals {
		if proposal.UniqueID() == id {
			return index, true
		}
	}

	return 0, false
}
