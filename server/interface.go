package server

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

//Client interface for mysterium centralized api - will be removed in the future
type Client interface {
	RegisterIdentity(identity identity.Identity, signer identity.Signer) (err error)
	NodeRegister(proposal dto_discovery.ServiceProposal) (err error)
	NodeSendStats(nodeKey string) (err error)
	FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error)
	SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error)
}
