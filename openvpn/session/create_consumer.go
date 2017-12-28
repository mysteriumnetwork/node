package session

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

type SessionCreateConsumer struct {
	CurrentProposalId   int
	SessionManager      session.ManagerInterface
	ClientConfigFactory func() *openvpn.ClientConfig
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

	clientConfig, err := consumer.newClientConfig()
	if err != nil {
		response = &SessionCreateResponse{
			Success: false,
			Message: "Failed to create VPN config.",
		}
		return
	}
	clientSession := consumer.newVpnSession(clientConfig)

	response = &SessionCreateResponse{
		Success: true,
		Session: clientSession,
	}
	return
}

func (consumer *SessionCreateConsumer) newClientConfig() (string, error) {
	vpnClientConfig := consumer.ClientConfigFactory()
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return "", err
	}

	return vpnClientConfigString, nil
}

func (consumer *SessionCreateConsumer) newVpnSession(vpnClientConfig string) SessionDto {
	sessionId := consumer.SessionManager.Create()

	return SessionDto{
		Id:     sessionId,
		Config: []byte(vpnClientConfig),
	}
}
