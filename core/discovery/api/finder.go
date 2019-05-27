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

package api

import (
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

// finderAPI implements ProposalFinder, which finds proposals from Mysterium API
type finderAPI struct {
	mysteriumAPI *mysterium.MysteriumAPI
}

// NewFinder creates new instance of finderAPI
func NewFinder(mysteriumAPI *mysterium.MysteriumAPI) *finderAPI {
	return &finderAPI{
		mysteriumAPI: mysteriumAPI,
	}
}

// GetProposal fetches service proposal from discovery by exact ID
func (finder *finderAPI) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	proposals, err := finder.mysteriumAPI.QueryProposals(mysterium.ProposalsQuery{
		NodeKey:     id.ProviderID,
		ServiceType: id.ServiceType,
	})
	if err != nil {
		return nil, err
	}
	if len(proposals) == 0 {
		return nil, nil
	}

	return &proposals[0], nil
}

// FindProposals fetches currently active service proposals from discovery
func (finder *finderAPI) FindProposals(filter market.ProposalFilter) ([]market.ServiceProposal, error) {
	return finder.mysteriumAPI.QueryProposals(mysterium.ProposalsQuery{
		NodeKey:            filter.ProviderID,
		ServiceType:        filter.ServiceType,
		AccessPolicyID:     filter.AccessPolicy.ID,
		AccessPolicySource: filter.AccessPolicy.Source,
	})
}
