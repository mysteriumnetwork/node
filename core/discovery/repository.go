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

package discovery

import (
	"sync"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/rs/zerolog/log"
)

// repository provides proposals from multiple other repositories.
type repository struct {
	delegates []proposal.Repository
}

// NewRepository constructs a new composite repository.
func NewRepository() *repository {
	return &repository{}
}

// Add adds a delegate repositories from which proposals can be acquired.
func (c *repository) Add(repository proposal.Repository) {
	c.delegates = append(c.delegates, repository)
}

// Proposal returns a single proposal by its ID.
func (c *repository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	allErrors := utils.ErrorCollection{}

	for _, delegate := range c.delegates {
		serviceProposal, err := delegate.Proposal(id)
		if err == nil {
			return serviceProposal, nil
		}
		allErrors.Add(err)
	}

	return nil, allErrors.Error()
}

// Proposals returns proposals matching the filter.
func (c *repository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	log.Debug().Msgf("Retrieving proposals from %d repositories", len(c.delegates))
	proposals := make([][]market.ServiceProposal, len(c.delegates))
	errors := make([]error, len(c.delegates))

	var wg sync.WaitGroup
	for i, delegate := range c.delegates {
		wg.Add(1)
		go func(idx int, repo proposal.Repository) {
			defer wg.Done()
			proposals[idx], errors[idx] = repo.Proposals(filter)
		}(i, delegate)
	}
	wg.Wait()

	uniqueProposals := make(map[market.ProposalID]market.ServiceProposal)
	for i, repoProposals := range proposals {
		log.Trace().Msgf("Retrieved %d proposals from repository %d", len(repoProposals), i)
		for _, p := range repoProposals {
			uniqueProposals[p.UniqueID()] = p
		}
	}

	var result []market.ServiceProposal
	for _, val := range uniqueProposals {
		result = append(result, val)
	}

	allErrors := utils.ErrorCollection{}
	allErrors.Add(errors...)

	log.Err(allErrors.Error()).Msgf("Returning %d unique proposals", len(result))
	return result, allErrors.Error()
}
