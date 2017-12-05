package session

import (
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

type SessionCreateHandler struct {
	CurrentProposalId   int
	SessionManager      session.ManagerInterface
	ClientConfigFactory func() *openvpn.ClientConfig
}

func (handler *SessionCreateHandler) GetRequestType() communication.RequestType {
	return SESSION_CREATE
}

func (handler *SessionCreateHandler) NewRequest() (requestPtr interface{}) {
	var request SessionCreateRequest
	return &request
}

func (handler *SessionCreateHandler) Handle(requestPtr interface{}) (response interface{}, err error) {
	request := requestPtr.(*SessionCreateRequest)
	if handler.CurrentProposalId != request.ProposalId {
		response = &SessionCreateResponse{
			Success: false,
			Message: fmt.Sprintf("Proposal doesn't exist: %d", request.ProposalId),
		}
		return
	}

	clientConfig, err := handler.newClientConfig()
	if err != nil {
		response = &SessionCreateResponse{
			Success: false,
			Message: "Failed to create VPN config.",
		}
		return
	}
	clientSession := handler.newVpnSession(clientConfig)

	response = &SessionCreateResponse{
		Success: true,
		Session: clientSession,
	}
	return
}

func (handler *SessionCreateHandler) newClientConfig() (string, error) {
	vpnClientConfig := handler.ClientConfigFactory()
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return "", err
	}

	return vpnClientConfigString, nil
}

func (handler *SessionCreateHandler) newVpnSession(vpnClientConfig string) VpnSession {
	sessionId := handler.SessionManager.Create()

	return VpnSession{
		Id:     sessionId,
		Config: vpnClientConfig,
	}
}
