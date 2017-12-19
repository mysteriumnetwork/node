package server

import (
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
    "github.com/mysterium/node/identity"
)

type Client interface {
	RegisterIdentity(identity identity.Identity) (err error)
	NodeRegister(proposal dto_discovery.ServiceProposal) (err error)
	NodeSendStats(nodeKey string, sessionStats []dto.SessionStats) (err error)
	SessionCreate(nodeKey string) (session dto.Session, err error)
	SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error)
}
