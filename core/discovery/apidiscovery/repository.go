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

package apidiscovery

import (
	"fmt"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

type apiRepository struct {
	discoveryAPI *mysterium.MysteriumAPI
}

// NewRepository constructs a new proposal repository (backed by API).
func NewRepository(api *mysterium.MysteriumAPI) *apiRepository {
	return &apiRepository{discoveryAPI: api}
}

// Proposal returns proposal by ID.
func (a *apiRepository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	proposals, err := a.discoveryAPI.QueryProposals(mysterium.ProposalsQuery{
		NodeKey:         id.ProviderID,
		ServiceType:     id.ServiceType,
		AccessPolicyAll: true,
	})
	if err != nil {
		return nil, err
	}
	if len(proposals) != 1 {
		return nil, fmt.Errorf("proposal does not exist: %+v", id)
	}
	return &proposals[0], nil
}

// Proposals returns proposals matching filter.
func (a *apiRepository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	return a.discoveryAPI.QueryProposals(filter.ToAPIQuery())
}
