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

package proposal

import (
	"github.com/mysteriumnetwork/node/core/discovery/reducer"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

// Filter defines all flags for proposal filtering in discovery of Mysterium Network
type Filter struct {
	ProviderID         string
	ServiceType        string
	LocationType       string
	AccessPolicyID     string
	AccessPolicySource string
}

// Matches return flag if filter matches given proposal
func (filter *Filter) Matches(proposal market.ServiceProposal) bool {
	conditions := make([]reducer.AndCondition, 0)
	if filter.ProviderID != "" {
		conditions = append(conditions, reducer.Equal(reducer.ProviderID, filter.ProviderID))
	}
	if filter.ServiceType != "" {
		conditions = append(conditions, reducer.Equal(reducer.ServiceType, filter.ServiceType))
	}
	if filter.LocationType != "" {
		conditions = append(conditions, reducer.Equal(reducer.LocationType, filter.LocationType))
	}
	if filter.AccessPolicyID != "" || filter.AccessPolicySource != "" {
		conditions = append(conditions, reducer.AccessPolicy(filter.AccessPolicyID, filter.AccessPolicySource))
	}
	if len(conditions) > 0 {
		return reducer.And(conditions...)(proposal)
	}
	return true
}

// ToAPIQuery serialises filter to query of Mysterium API
func (filter *Filter) ToAPIQuery() mysterium.ProposalsQuery {
	return mysterium.ProposalsQuery{
		NodeKey:            filter.ProviderID,
		ServiceType:        filter.ServiceType,
		AccessPolicyID:     filter.AccessPolicyID,
		AccessPolicySource: filter.AccessPolicySource,
	}
}
