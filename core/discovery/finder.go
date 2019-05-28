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
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

// ProposalReducer proposal match function
type ProposalReducer func(proposal market.ServiceProposal) bool

// ProposalFilter defines interface with proposal match function
type ProposalFilter interface {
	Matches(proposal market.ServiceProposal) bool
	ToAPIQuery() mysterium.ProposalsQuery
}

// Finder finds proposals from local storage
type Finder struct {
	storage *ProposalStorage
}

// NewFinder creates instance of local storage Finder
func NewFinder(storage *ProposalStorage) *Finder {
	return &Finder{storage: storage}
}

// GetProposal fetches service proposal from discovery by exact ID
func (finder *Finder) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	for _, proposal := range finder.storage.Proposals() {
		if proposal.UniqueID() == id {
			return &proposal, nil
		}
	}

	return nil, nil
}

// MatchProposals fetches currently active service proposals from discovery by match function
func (finder *Finder) MatchProposals(match ProposalReducer) ([]market.ServiceProposal, error) {
	proposalsFiltered := make([]market.ServiceProposal, 0)
	for _, proposal := range finder.storage.Proposals() {
		if match(proposal) {
			proposalsFiltered = append(proposalsFiltered, proposal)
		}
	}
	return proposalsFiltered, nil
}

// FindProposals fetches currently active service proposals from discovery by given filter
func (finder *Finder) FindProposals(filter ProposalFilter) ([]market.ServiceProposal, error) {
	return finder.MatchProposals(filter.Matches)
}
