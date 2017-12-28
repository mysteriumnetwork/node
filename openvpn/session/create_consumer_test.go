package session

import (
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var consumer = SessionCreateConsumer{
	CurrentProposalId: 101,
	SessionManager:    &session.ManagerFake{},
}

func TestConsumer_UnknownProposal(t *testing.T) {
	request := consumer.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 100
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: false,
			Message: "Proposal doesn't exist: 100",
		},
		sessionResponse,
	)
}

func TestConsumer_Success(t *testing.T) {
	request := consumer.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 101
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: true,
			Session: SessionDto{
				Id:     "new-id",
				Config: []byte("new-config"),
			},
		},
		sessionResponse,
	)
}
