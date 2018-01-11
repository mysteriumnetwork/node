package server

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

//Client interface for mysterium centralized api - will be removed in the future
type Client interface {
	FindProposals(nodeKey string) (proposals []dto_discovery.ServiceProposal, err error)
	//these functions have side effects - therefore signer required
	RegisterIdentity(identity identity.Identity, signer identity.Signer) (err error)
	RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	NodeSendStats(nodeKey string, signer identity.Signer) (err error)
	SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error)
}
