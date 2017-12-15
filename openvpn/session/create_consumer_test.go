package session

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

var sessionManager = session.Manager{
	Generator: &session.GeneratorFake{
		SessionIdMock: session.SessionId("session-mock"),
	},
}

var consumer = SessionCreateConsumer{
	CurrentProposalId: 101,
	SessionManager:    &sessionManager,
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
	consumer.ClientConfigFactory = func() *openvpn.ClientConfig {
		clientConfig := openvpn.ClientConfig{&openvpn.Config{}}
		clientConfig.SetPort(1000)
		return &clientConfig
	}

	request := consumer.NewRequest().(*SessionCreateRequest)
	request.ProposalId = 101
	sessionResponse, err := consumer.Consume(request)

	assert.NoError(t, err)
	assert.Exactly(
		t,
		&SessionCreateResponse{
			Success: true,
			Session: VpnSession{
				Id:     "session-mock",
				Config: "port 1000\n",
			},
		},
		sessionResponse,
	)
}
