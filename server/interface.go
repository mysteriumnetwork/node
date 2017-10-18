package server

import (
	"github.com/mysterium/node/server/dto"
	dto2 "github.com/mysterium/node/service_discovery/dto"
)

type Client interface {
	NodeRegister(proposal dto2.ServiceProposal) (err error)
	NodeSendStats(nodeKey string, sessionStats []dto.SessionStats) (err error)
	SessionCreate(nodeKey string) (session dto.Session, err error)
	SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error)
}
