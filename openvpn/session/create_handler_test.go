package session

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var sessionManager = session.Manager{
	Generator: &session.GeneratorMock{},
}

var clientConfig = openvpn.ClientConfig{&openvpn.Config{}}

var handler = SessionCreateHandler{
	CurrentProposalId: 101,
	SessionManager:    &sessionManager,
	ClientConfigFactory: func() *openvpn.ClientConfig {
		return &clientConfig
	},
}

func TestHandler_UnknownProposal(t *testing.T) {
	request := handler.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 100
	sessionResponse, err := handler.Handle(request)

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

func TestHandler_Success(t *testing.T) {
	request := handler.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 101
	sessionResponse, err := handler.Handle(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: true,
			Session: VpnSession{
				Id:     "",
				Config: "",
			},
		},
		sessionResponse,
	)
}
