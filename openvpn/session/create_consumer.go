package session

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/session"
)

type SessionCreateConsumer struct {
	CurrentProposalId int
	SessionManager    session.ManagerInterface
}

func (consumer *SessionCreateConsumer) GetRequestType() communication.RequestType {
	return SESSION_CREATE
}

func (consumer *SessionCreateConsumer) NewRequest() (requestPtr interface{}) {
	var request SessionCreateRequest
	return &request
}

func (consumer *SessionCreateConsumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	request := requestPtr.(*SessionCreateRequest)
	if consumer.CurrentProposalId != request.ProposalId {
		response = &SessionCreateResponse{
			Success: false,
			Message: fmt.Sprintf("Proposal doesn't exist: %d", request.ProposalId),
		}
		return
	}

	clientSession, err := consumer.SessionManager.Create()
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
			Id:     clientSession.Id,
			Config: []byte(clientSession.Config),
		},
	}
	return
}
