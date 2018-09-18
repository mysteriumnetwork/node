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
	"context"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server/dto"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

// Client is interface how to access Mysterium API
type Client interface {
	RegisterIdentity(ctx context.Context, id identity.Identity, signer identity.Signer) (err error)

	FindProposals(ctx context.Context, providerID string) (proposals []dto_discovery.ServiceProposal, err error)
	RegisterProposal(ctx context.Context, proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	UnregisterProposal(ctx context.Context, proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	PingProposal(ctx context.Context, proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)

	SendSessionStats(sessionId session.SessionID, sessionStats dto.SessionStats, signer identity.Signer) (err error)
}
