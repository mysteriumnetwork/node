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
	"fmt"
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
func (s *ProposalStorage) Proposals() []market.ServiceProposal {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.proposals
}

// MatchProposals fetches currently active service proposals from storage by match function
func (s *ProposalStorage) MatchProposals(match ProposalReducer) ([]market.ServiceProposal, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	proposals := make([]market.ServiceProposal, 0)
	for _, proposal := range s.proposals {
		if match(proposal) {
			proposals = append(proposals, proposal)
		}
	}
	return proposals, nil
}

// FindProposals fetches currently active service proposals from storage by given filter
func (s *ProposalStorage) FindProposals(filter ProposalFilter) ([]market.ServiceProposal, error) {
	return s.MatchProposals(filter.Matches)
}

// Set puts given list proposals to storage
func (s *ProposalStorage) Set(proposals []market.ServiceProposal) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.proposals = proposals
}

// HasProposal checks if proposal exists in storage
func (s *ProposalStorage) HasProposal(id market.ProposalID) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exist := s.getProposalIndex(id)
	return exist
}

// GetProposal returns proposal from storage
func (s *ProposalStorage) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	index, exist := s.getProposalIndex(id)
	if !exist {
		return nil, fmt.Errorf(`proposal does not exist: %v`, id)
	}
	return &s.proposals[index], nil
}

// AddProposal appends given proposal to storage
func (s *ProposalStorage) AddProposal(proposal market.ServiceProposal) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exist := s.getProposalIndex(proposal.UniqueID()); exist {
		return
	}
	s.proposals = append(s.proposals, proposal)
}

// RemoveProposal removes proposal from storage
func (s *ProposalStorage) RemoveProposal(id market.ProposalID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if index, exist := s.getProposalIndex(id); exist {
		s.proposals = append(s.proposals[:index], s.proposals[index+1:]...)
	}
}

func (s *ProposalStorage) getProposalIndex(id market.ProposalID) (int, bool) {
	for index, proposal := range s.proposals {
		if proposal.UniqueID() == id {
			return index, true
		}
	}

	return 0, false
}
