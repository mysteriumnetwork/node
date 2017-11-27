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

func TestHandler_UnknownProposal(t *testing.T) {
	handler := CreateResponseHandler{
		ProposalId:     101,
		SessionManager: &sessionManager,
		ClientConfigFactory: func() *openvpn.ClientConfig {
			config := &openvpn.ClientConfig{&openvpn.Config{}}
			config.SetPort(1001)
			return config
		},
	}

	sessionResponse := handler.Handle("100")
	assert.JSONEq(
		t,
		`{
			"success": false,
			"message": "Proposal doesn't exist.",
			"session": {
				"id": "",
				"config": ""
			}
		}`,
		sessionResponse,
	)
}

func TestHandler_InvalidProposalId(t *testing.T) {
	handler := CreateResponseHandler{
		ProposalId:     101,
		SessionManager: &sessionManager,
		ClientConfigFactory: func() *openvpn.ClientConfig {
			config := &openvpn.ClientConfig{&openvpn.Config{}}
			config.SetPort(1001)
			return config
		},
	}

	sessionResponse := handler.Handle("abc")
	assert.JSONEq(
		t,
		`{
			"success": false,
			"message": "Invalid Proposal ID format. Expected int.",
			"session": {
				"id": "",
				"config": ""
			}
		}`,
		sessionResponse,
	)
}

func TestHandler_Success(t *testing.T) {
	handler := CreateResponseHandler{
		ProposalId:     101,
		SessionManager: &sessionManager,
		ClientConfigFactory: func() *openvpn.ClientConfig {
			config := &openvpn.ClientConfig{&openvpn.Config{}}
			config.SetPort(1001)
			return config
		},
	}

	sessionResponse := handler.Handle("101")
	assert.JSONEq(
		t,
		`{
			"success": true,
			"message": "",
			"session": {
				"id": "",
				"config": "port 1001\n"
			}
		}`,
		sessionResponse,
	)
}
