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
	proposalsMock []dto_discovery.ServiceProposal
}

func (client *ClientFake) NodeRegister(proposal dto_discovery.ServiceProposal) (err error) {
	client.proposalsMock = append(client.proposalsMock, proposal)
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake node registered: ", proposal)

	return nil
}

func (client *ClientFake) RegisterIdentity(identity identity.Identity) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake identity registered: ", identity)

	return nil
}

func (client *ClientFake) NodeSendStats(nodeKey string, sessionStats []dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Node stats sent: ", nodeKey)

	return nil
}

func (client *ClientFake) FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake proposals requested for node_key: ", nodeKey)

	for _, proposal := range client.proposalsMock {
		var filterMatched = true
		if nodeKey != "" {
			filterMatched = filterMatched && (nodeKey == proposal.ProviderId)
		}
		if filterMatched {
			proposals = append(proposals, proposal)
		}
	}

	return proposals, nil
}

func (client *ClientFake) SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Session stats sent: ", sessionId)

	return nil
}
