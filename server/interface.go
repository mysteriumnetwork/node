/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package server

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server/dto"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

// Client is interface how to access Mysterium API
type Client interface {
	RegisterIdentity(id identity.Identity, signer identity.Signer) (err error)

	FindProposals(providerID string) (proposals []dto_discovery.ServiceProposal, err error)
	RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	UnregisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	PingProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)

	SendSessionStats(sessionId session.ID, sessionStats dto.SessionStats, signer identity.Signer) (err error)
}

// MorqaClient allows to interact with a quality oracle service (MORQA)
type MorqaClient interface {
	ProposalsQuality() ([]json.RawMessage, error)
}
