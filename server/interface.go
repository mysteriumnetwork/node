package server

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

type Client interface {
	RegisterIdentity(identity identity.Identity) (err error)
	NodeRegister(proposal dto_discovery.ServiceProposal) (err error)
	NodeSendStats(nodeKey string, sessionStats []dto.SessionStatsDeprecated) (err error)
	FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error)
	SendSessionStats(sessionId string, sessionStats dto.SessionStats) (err error)
}
