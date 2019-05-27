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

package market

var (
	emptyFilterLocation     = LocationFilter{}
	emptyFilterAccessPolicy = AccessPolicyFilter{}
)

// ProposalFilter defines all flags for proposal filtering in discovery of Mysterium Network
type ProposalFilter struct {
	ProviderID   string
	ServiceType  string
	Location     LocationFilter
	AccessPolicy AccessPolicyFilter
}

// Matches return flag if filter matches given proposal
func (filter *ProposalFilter) Matches(proposal ServiceProposal) bool {
	if filter.ProviderID != "" && filter.ProviderID != proposal.ProviderID {
		return false
	}
	if filter.ServiceType != "" && filter.ServiceType != proposal.ServiceType {
		return false
	}
	if filter.Location != emptyFilterLocation && !filter.Location.Matches(proposal) {
		return false
	}
	if filter.AccessPolicy != emptyFilterAccessPolicy && !filter.AccessPolicy.Matches(proposal) {
		return false
	}
	return true
}

// LocationFilter defines flags for proposal location filtering
type LocationFilter struct {
	NodeType string
}

// Matches return flag if filter matches given proposal
func (filter *LocationFilter) Matches(proposal ServiceProposal) bool {
	location := proposal.ServiceDefinition.GetLocation()
	if filter.NodeType != "" && filter.NodeType != location.NodeType {
		return false
	}

	return true
}

// AccessPolicyFilter defines flags for proposal acccess policy filtering
type AccessPolicyFilter struct {
	ID     string
	Source string
}

// Matches return flag if filter matches given proposal
func (filter *AccessPolicyFilter) Matches(proposal ServiceProposal) bool {
	// These proposals accepts all access lists
	if proposal.AccessPolicies == nil {
		return false
	}

	var match bool
	for _, policy := range *proposal.AccessPolicies {
		if filter.ID != "" {
			match = filter.ID == policy.ID
		}
		if filter.Source != "" {
			match = match && filter.Source == policy.Source
		}
		if match == true {
			break
		}
	}
	return match
}
