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

var (
	provider1            = "0x1"
	provider2            = "0x2"
	serviceTypeStreaming = "streaming"
	serviceTypeNoop      = "noop"
	accessRuleWhitelist  = market.AccessPolicy{
		ID:     "whitelist",
		Source: "whitelist.txt",
	}
	accessRuleBlacklist = market.AccessPolicy{
		ID:     "blacklist",
		Source: "blacklist.txt",
	}
	locationDatacenter  = market.Location{ASN: 1000, Country: "DE", City: "Berlin", IPType: "datacenter"}
	locationResidential = market.Location{ASN: 124, Country: "LT", City: "Vilnius", IPType: "residential"}

	proposalEmpty              = market.NewProposal("", serviceTypeNoop, market.NewProposalOpts{})
	proposalProvider1Streaming = market.NewProposal(provider1, serviceTypeStreaming, market.NewProposalOpts{
		Location:       &locationDatacenter,
		AccessPolicies: []market.AccessPolicy{accessRuleWhitelist},
	})
	proposalProvider1Noop      = market.NewProposal(provider1, serviceTypeNoop, market.NewProposalOpts{})
	proposalProvider2Streaming = market.NewProposal(provider2, serviceTypeStreaming, market.NewProposalOpts{
		Location:       &locationResidential,
		AccessPolicies: []market.AccessPolicy{accessRuleWhitelist, accessRuleBlacklist},
	})
)

func conditionAlwaysMatch(_ market.ServiceProposal) bool {
	return true
}

func conditionNeverMatch(_ market.ServiceProposal) bool {
	return false
}

func conditionIsProvider1(provider market.ServiceProposal) bool {
	return provider.ProviderID == provider1
}

func conditionIsStreaming(provider market.ServiceProposal) bool {
	return provider.ServiceType == "streaming"
}

func fieldCompatibility(proposal market.ServiceProposal) interface{} {
	return proposal.Compatibility
}

func fieldProviderID(proposal market.ServiceProposal) interface{} {
	return proposal.ProviderID
}
