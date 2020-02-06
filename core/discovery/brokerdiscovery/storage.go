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

package brokerdiscovery

import (
	"fmt"
	"sync"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
)

// Worker continuously tracks service proposals from discovery service to storage
type Worker interface {
	Start() error
	Stop()
}

// ProposalReducer proposal match function
type ProposalReducer func(proposal market.ServiceProposal) bool

// NewStorage creates new instance of ProposalStorage
func NewStorage(eventPublisher eventbus.Publisher) *ProposalStorage {
	return &ProposalStorage{
		eventPublisher: eventPublisher,
		proposals:      make([]market.ServiceProposal, 0),
	}
}

// ProposalStorage represents table of currently active proposals in Mysterium Discovery
type ProposalStorage struct {
	eventPublisher eventbus.Publisher

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
	for _, p := range s.proposals {
		if match(p) {
			proposals = append(proposals, p)
		}
	}
	return proposals, nil
}

// FindProposals fetches currently active service proposals from storage by given filter
func (s *ProposalStorage) FindProposals(filter proposal.Filter) ([]market.ServiceProposal, error) {
	return s.MatchProposals(filter.Matches)
}

// Set puts given list proposals to storage
func (s *ProposalStorage) Set(proposals []market.ServiceProposal) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	proposalsOld := s.proposals
	for _, p := range proposals {
		index, exist := s.getProposalIndex(proposalsOld, p.UniqueID())
		if exist {
			go s.eventPublisher.Publish(discovery.AppTopicProposalUpdated, p)
			proposalsOld = append(proposalsOld[:index], proposalsOld[index+1:]...)
		} else {
			go s.eventPublisher.Publish(discovery.AppTopicProposalAdded, p)
		}
	}
	for _, p := range proposalsOld {
		go s.eventPublisher.Publish(discovery.AppTopicProposalRemoved, p)
	}
	s.proposals = proposals
}

// HasProposal checks if proposal exists in storage
func (s *ProposalStorage) HasProposal(id market.ProposalID) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exist := s.getProposalIndex(s.proposals, id)
	return exist
}

// GetProposal returns proposal from storage
func (s *ProposalStorage) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	index, exist := s.getProposalIndex(s.proposals, id)
	if !exist {
		return nil, fmt.Errorf(`proposal does not exist: %v`, id)
	}
	return &s.proposals[index], nil
}

// AddProposal appends given proposal to storage
func (s *ProposalStorage) AddProposal(proposals ...market.ServiceProposal) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, p := range proposals {
		if index, exist := s.getProposalIndex(s.proposals, p.UniqueID()); !exist {
			s.eventPublisher.Publish(discovery.AppTopicProposalAdded, p)
			s.proposals = append(s.proposals, p)
		} else {
			s.eventPublisher.Publish(discovery.AppTopicProposalUpdated, p)
			s.proposals[index] = p
		}
	}
}

// RemoveProposal removes proposal from storage
func (s *ProposalStorage) RemoveProposal(id market.ProposalID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if index, exist := s.getProposalIndex(s.proposals, id); exist {
		go s.eventPublisher.Publish(discovery.AppTopicProposalRemoved, s.proposals[index])
		s.proposals = append(s.proposals[:index], s.proposals[index+1:]...)
	}
}

func (s *ProposalStorage) getProposalIndex(proposals []market.ServiceProposal, id market.ProposalID) (int, bool) {
	for index, p := range proposals {
		if p.UniqueID() == id {
			return index, true
		}
	}

	return 0, false
}
