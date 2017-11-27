package session

import (
	"encoding/json"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"strconv"
)

type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session VpnSession `json:"session"`
}

type ClientConfigFactory func() *openvpn.ClientConfig

type CreateResponseHandler struct {
	ProposalId          int
	SessionManager      session.ManagerInterface
	ClientConfigFactory ClientConfigFactory
}

func (scr *CreateResponseHandler) Handle(proposalId string) string {
	response := SessionCreateResponse{}

	numericProposalId, err := strconv.Atoi(proposalId)
	if err != nil {
		response.Success = false
		response.Message = "Invalid Proposal ID format. Expected int."

		return serializeCreateResponse(response)
	}

	if scr.ProposalId != numericProposalId {
		response.Success = false
		response.Message = "Proposal doesn't exist."

		return serializeCreateResponse(response)
	}

	str := serializeCreateResponse(scr.buildResponse())
	return str
}

func (scr *CreateResponseHandler) buildResponse() (response SessionCreateResponse) {
	vpnClientConfig := scr.ClientConfigFactory()
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		response.Success = false
		response.Message = "Failed to create VPN vpnClientConfigString."
		return
	}

	response.Success = true
	response.Session = NewVpnSession(scr.SessionManager, vpnClientConfigString)

	return
}

func serializeCreateResponse(response SessionCreateResponse) string {
	scr, err := json.Marshal(response)
	if err != nil {
		return "Error serializing response."
	}

	return string(scr)
}
