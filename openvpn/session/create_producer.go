package session

import (
	"github.com/mysterium/node/communication"
	"github.com/pkg/errors"
)

type SessionCreateProducer struct {
	ProposalId int
}

func (producer *SessionCreateProducer) GetRequestType() communication.RequestType {
	return SESSION_CREATE
}

func (producer *SessionCreateProducer) NewResponse() (responsePtr interface{}) {
	var response SessionCreateResponse
	return &response
}

func (producer *SessionCreateProducer) Produce() (requestPtr interface{}) {
	return &SessionCreateRequest{
		ProposalId: producer.ProposalId,
	}
}

func RequestSessionCreate(sender communication.Sender, proposalId int) (*VpnSession, error) {
	responsePtr, err := sender.Request(&SessionCreateProducer{
		ProposalId: proposalId,
	})
	response := responsePtr.(*SessionCreateResponse)

	if err != nil || !response.Success {
		return nil, errors.New("Session create failed. " + response.Message)
	}

	return &response.Session, nil
}
