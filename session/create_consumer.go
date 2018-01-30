package session

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

type SessionCreateConsumer struct {
	CurrentProposalID int
	SessionManager    Manager
	PeerID            identity.Identity
}

func (consumer *SessionCreateConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (consumer *SessionCreateConsumer) NewRequest() (requestPtr interface{}) {
	var request SessionCreateRequest
	return &request
}

func (consumer *SessionCreateConsumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	request := requestPtr.(*SessionCreateRequest)
	if consumer.CurrentProposalID != request.ProposalId {
		response = &SessionCreateResponse{
			Success: false,
			Message: fmt.Sprintf("Proposal doesn't exist: %d", request.ProposalId),
		}
		return
	}

	clientSession, err := consumer.SessionManager.Create(consumer.PeerID)
	if err != nil {
		response = &SessionCreateResponse{
			Success: false,
			Message: "Failed to create session.",
		}
		return
	}

	response = &SessionCreateResponse{
		Success: true,
		Session: SessionDto{
			ID:     clientSession.ID,
			Config: clientSession.Config,
		},
	}
	return
}
