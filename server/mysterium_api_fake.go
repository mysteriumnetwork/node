package server

import (
	"github.com/mysterium/node/server/dto"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewClientFake() *ClientFake {
	return &ClientFake{
		proposalsMock: make([]dto_discovery.ServiceProposal, 0),
	}
}

type ClientFake struct {
	RegisteredIdentity identity.Identity
	proposalsMock      []dto_discovery.ServiceProposal
}

func (client *ClientFake) RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error) {
	client.proposalsMock = append(client.proposalsMock, proposal)
	log.Info(mysteriumAPILogPrefix, "Fake node registered: ", proposal)

	return nil
}

func (client *ClientFake) RegisterIdentity(newIdentity identity.Identity, signer identity.Signer) (err error) {
	client.RegisteredIdentity = newIdentity
	log.Info(mysteriumAPILogPrefix, "Fake newIdentity registered: ", newIdentity)

	return nil
}

func (client *ClientFake) NodeSendStats(nodeKey string, signer identity.Signer) (err error) {
	log.Info(mysteriumAPILogPrefix, "Node stats sent: ", nodeKey)

	return nil
}

func (client *ClientFake) FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error) {
	log.Info(mysteriumAPILogPrefix, "Fake proposals requested for node_key: ", nodeKey)

	for _, proposal := range client.proposalsMock {
		var filterMatched = true
		if nodeKey != "" {
			filterMatched = filterMatched && (nodeKey == proposal.ProviderID)
		}
		if filterMatched {
			proposals = append(proposals, proposal)
		}
	}

	return proposals, nil
}

func (client *ClientFake) SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error) {
	log.Info(mysteriumAPILogPrefix, "Session stats sent: ", sessionId)

	return nil
}
