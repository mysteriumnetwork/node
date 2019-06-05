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

package reducer

import (
	"github.com/mysteriumnetwork/node/market"
)

// ProviderID selects provider id value from proposal
func ProviderID(proposal market.ServiceProposal) interface{} {
	return proposal.ProviderID
}

// ServiceType selects service type from proposal
func ServiceType(proposal market.ServiceProposal) interface{} {
	return proposal.ServiceType
}

// Service selects service definition from proposal
func Service(proposal market.ServiceProposal) interface{} {
	return proposal.ServiceDefinition
}

// Location selects service location from proposal
func Location(proposal market.ServiceProposal) interface{} {
	service := proposal.ServiceDefinition
	if service == nil {
		return nil
	}
	return service.GetLocation()
}

// LocationCountry selects location country from proposal
func LocationCountry(proposal market.ServiceProposal) interface{} {
	service := proposal.ServiceDefinition
	if service == nil {
		return nil
	}
	return service.GetLocation().Country
}

// LocationType selects location type from proposal
func LocationType(proposal market.ServiceProposal) interface{} {
	service := proposal.ServiceDefinition
	if service == nil {
		return nil
	}
	return service.GetLocation().NodeType
}

// AccessPolicy returns a matcher for checking if proposal allows given access policy
func AccessPolicy(id, source string) func(market.ServiceProposal) bool {
	return func(proposal market.ServiceProposal) bool {
		// These proposals accepts all access lists
		if proposal.AccessPolicies == nil {
			return false
		}

		var match bool
		for _, policy := range *proposal.AccessPolicies {
			match = (id == "" || policy.ID == id) &&
				(source == "" || policy.Source == source)
			if match {
				break
			}
		}
		return match
	}
}
