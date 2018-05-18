package session

import (
	"errors"
	"github.com/mysterium/node/communication"
)

type createProducer struct {
	ProposalID int
}

func (producer *createProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (producer *createProducer) NewResponse() (responsePtr interface{}) {
	return &SessionCreateResponse{}
}

func (producer *createProducer) Produce() (requestPtr interface{}) {
	return &SessionCreateRequest{
		ProposalId: producer.ProposalID,
	}
}

// RequestSessionCreate requests session creation and returns session DTO
func RequestSessionCreate(sender communication.Sender, proposalID int) (*SessionDto, error) {
	responsePtr, err := sender.Request(&createProducer{
		ProposalID: proposalID,
	})
	response := responsePtr.(*SessionCreateResponse)

	if err != nil || !response.Success {
		return nil, errors.New("SessionDto create failed. " + response.Message)
	}

	return &response.Session, nil
}
