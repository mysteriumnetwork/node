package server

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

// Client is interface how to access Mysterium API
type Client interface {
	RegisterIdentity(id identity.Identity, signer identity.Signer) (err error)

	FindProposals(providerID string) (proposals []dto_discovery.ServiceProposal, err error)
	RegisterProposal(proposal dto_discovery.ServiceProposal, signer identity.Signer) (err error)
	SendProposalStats(providerID string, signer identity.Signer) (err error)

	SendSessionStats(sessionId string, sessionStats dto.SessionStats, signer identity.Signer) (err error)
}
