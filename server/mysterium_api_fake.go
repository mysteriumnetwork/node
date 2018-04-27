package server

import (
	"github.com/mysterium/node/server/dto"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
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
func (client *ClientFake) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error) {
	log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionId)

	return nil
}
