package session

import (
	"github.com/mysterium/node/communication"
	"github.com/pkg/errors"
)

type SessionCreateProducer struct {
	ProposalId int
}

func (producer *SessionCreateProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *SessionCreateProducer) NewResponse() (responsePtr interface{}) {
	var response SessionCreateResponse
	return &response
}

func (producer *SessionCreateProducer) Produce() (request interface{}) {
	return SessionCreateRequest{
		ProposalId: producer.ProposalId,
	}
}

func RequestSessionCreate(sender communication.Sender, proposalId int) (*SessionDto, error) {
	responsePtr, err := sender.Request(&SessionCreateProducer{
		ProposalId: proposalId,
	})
	response := responsePtr.(*SessionCreateResponse)

	if err != nil || !response.Success {
		return nil, errors.New("SessionDto create failed. " + response.Message)
	}

	return &response.Session, nil
}
