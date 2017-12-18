package server

import (
	"github.com/mysterium/node/server/dto"

	"fmt"
	log "github.com/cihub/seelog"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewClientFake() Client {
	return &clientFake{
		proposalsByProvider: make(map[string]dto_discovery.ServiceProposal, 0),
	}
}

type clientFake struct {
	proposalsByProvider map[string]dto_discovery.ServiceProposal
}

func (client *clientFake) NodeRegister(proposal dto_discovery.ServiceProposal) (err error) {
	client.proposalsByProvider[string(proposal.ProviderId)] = proposal
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake node registered: ", proposal)

	return nil
}

func (client *clientFake) RegisterIdentity(identity dto_discovery.Identity) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake identity registered: ", identity)

	return nil
}

func (client *clientFake) NodeSendStats(nodeKey string, sessionStats []dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Node stats sent: ", nodeKey)

	return nil
}

func (client *clientFake) SessionCreate(nodeKey string) (session dto.Session, err error) {
	if proposal, ok := client.proposalsByProvider[nodeKey]; ok {
		session = dto.Session{
			Id:              nodeKey + "-session",
			ServiceProposal: proposal,
		}
		return
	}

	err = fmt.Errorf("Fake node not found: %s", nodeKey)
	return
}

func (client *clientFake) SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Session stats sent: ", sessionId)

	return nil
}
