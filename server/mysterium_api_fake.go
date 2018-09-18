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
	log "github.com/cihub/seelog"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server/dto"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

// NewClientFake constructs fake API client
func NewClientFake() *ClientFake {
	return &ClientFake{
		proposalsMock: make([]dto_discovery.ServiceProposal, 0),
	}
}

// ClientFake is fake client to Mysterium API
type ClientFake struct {
	RegisteredIdentity identity.Identity
	proposalsMock      []dto_discovery.ServiceProposal
}

// RegisterProposal announces service proposal
func (client *ClientFake) RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error) {
	client.proposalsMock = append(client.proposalsMock, proposal)
	log.Info(mysteriumAPILogPrefix, "Fake proposal registered: ", proposal)

	return nil
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (client *ClientFake) UnregisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) error {
	remainingProposals := make([]dto_discovery.ServiceProposal, 0)

	for _, pr := range client.proposalsMock {
		if proposal.ProviderID != pr.ProviderID {
			remainingProposals = append(remainingProposals, proposal)
		}
	}
	client.proposalsMock = remainingProposals

	log.Info(mysteriumAPILogPrefix, "Fake proposal unregistered: ", proposal)

	return nil
}

// RegisterIdentity announces that new identity was created
func (client *ClientFake) RegisterIdentity(id identity.Identity, signer identity.Signer) (err error) {
	client.RegisteredIdentity = id
	log.Info(mysteriumAPILogPrefix, "Fake newIdentity registered: ", id.Address)

	return nil
}

// PingProposal heartbeats that service proposal is still active
func (client *ClientFake) PingProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error) {
	log.Info(mysteriumAPILogPrefix, "Proposal stats sent: ", proposal.ProviderID)

	return nil
}

// FindProposals fetches announced proposals by given filters
func (client *ClientFake) FindProposals(providerID string) (proposals []dto_discovery.ServiceProposal, err error) {
	log.Info(mysteriumAPILogPrefix, "Fake proposals requested for provider: ", providerID)

	for _, proposal := range client.proposalsMock {
		var filterMatched = true
		if providerID != "" {
			filterMatched = filterMatched && (providerID == proposal.ProviderID)
		}
		if filterMatched {
			proposals = append(proposals, proposal)
		}
	}

	return proposals, nil
}

// SendSessionStats heartbeats that session is still active + session upload and download amounts
func (client *ClientFake) SendSessionStats(sessionId session.SessionID, sessionStats dto.SessionStats, signer identity.Signer) (err error) {
	log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionId)

	return nil
}
