/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// ProposalRegistry defines methods for proposal lifecycle - registration, keeping up to date, removal
type ProposalRegistry interface {
	RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error
	PingProposal(proposal market.ServiceProposal, signer identity.Signer) error
	UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error
}

// ProposalReducer proposal match function
type ProposalReducer func(proposal market.ServiceProposal) bool

// ProposalMatcher defines interface to fetch currently active service proposals from discovery by match function
type ProposalMatcher interface {
	MatchProposals(match ProposalReducer) ([]market.ServiceProposal, error)
}
