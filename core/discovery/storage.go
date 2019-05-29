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

	"github.com/mysteriumnetwork/node/market"
)

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

// Proposals returns table of proposals in storage
func (storage *ProposalStorage) Proposals() []market.ServiceProposal {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	return storage.proposals
}

// Set puts given proposal to storage
func (storage *ProposalStorage) Set(proposals ...market.ServiceProposal) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	storage.proposals = proposals
}
