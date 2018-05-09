package session

import (
	"github.com/mysterium/node/communication"
)

const endpointSessionCreate = communication.RequestEndpoint("session-create")

//SessionCreateRequest structure represents message from service consumer to initiate session for given proposal id
type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

//SessionCreateResponse structure represents service provider response to given session request from consumer
type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session SessionDto `json:"session"`
}

//SessionDto structure represents session information data within session creation response (session id and configuration options for underlaying service type)
type SessionDto struct {
	ID     SessionID `json:"id"`
	Config VPNConfig `json:"config"`
}

//VPNConfig structure represents VPN configuration options for given session
type VPNConfig struct {
	RemoteIP        string `json:"remote"`
	RemotePort      int    `json:"port"`
	RemoteProtocol  string `json:"protocol"`
	TLSPresharedKey string `json:"TLSPresharedKey"`
	CACertificate   string `json:"CACertificate"`
}
