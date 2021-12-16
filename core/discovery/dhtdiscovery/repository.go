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

package dhtdiscovery

import (
	"sync"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
)

// Repository provides proposals from the DHT.
type Repository struct {
	stopOnce sync.Once
	stopChan chan struct{}
}

// NewRepository constructs a new proposal repository (backed by the DHT).
func NewRepository() *Repository {
	return &Repository{
		stopChan: make(chan struct{}),
	}
}

// Proposal returns a single proposal by its ID.
func (r *Repository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	return nil, nil
}

// Proposals returns proposals matching the filter.
func (r *Repository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return []market.ServiceProposal{}, nil
}

// Countries returns proposals per country matching the filter.
func (r *Repository) Countries(filter *proposal.Filter) (map[string]int, error) {
	return nil, nil
}

// Start begins proposals synchronization to storage.
func (r *Repository) Start() error {
	return nil
}

// Stop ends proposals synchronization to storage.
func (r *Repository) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopChan)
	})
}
