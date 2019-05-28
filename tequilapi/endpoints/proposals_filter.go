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

package endpoints

import (
	"github.com/mysteriumnetwork/node/core/discovery/reducer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

// proposalsFilter defines all flags for proposal filtering in discovery of Mysterium Network
type proposalsFilter struct {
	providerID         string
	serviceType        string
	locationType       string
	accessPolicyID     string
	accessPolicySource string
}

// Matches return flag if filter matches given proposal
func (filter *proposalsFilter) Matches(proposal market.ServiceProposal) bool {
	conditions := make([]reducer.AndCondition, 0)
	if filter.providerID != "" {
		conditions = append(conditions, reducer.Equal(reducer.ProviderID, filter.providerID))
	}
	if filter.serviceType != "" {
		conditions = append(conditions, reducer.Equal(reducer.ServiceType, filter.serviceType))
	}
	if filter.locationType != "" {
		conditions = append(conditions, reducer.Equal(reducer.LocationType, filter.locationType))
	}
	if filter.accessPolicyID != "" || filter.accessPolicySource != "" {
		conditions = append(conditions, reducer.AccessPolicy(filter.accessPolicyID, filter.accessPolicySource))
	}
	if len(conditions) > 0 {
		return reducer.And(conditions...)(proposal)
	}

	return reducer.All()(proposal)
}

// ToAPIQuery serialises filter to query of Mysterium API
func (filter *proposalsFilter) ToAPIQuery() mysterium.ProposalsQuery {
	return mysterium.ProposalsQuery{
		NodeKey:            filter.providerID,
		ServiceType:        filter.serviceType,
		AccessPolicyID:     filter.accessPolicyID,
		AccessPolicySource: filter.accessPolicySource,
	}
}
