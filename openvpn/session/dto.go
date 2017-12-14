package session

import "github.com/mysterium/node/communication"

const SESSION_CREATE = communication.RequestType("session-create")

type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session VpnSession `json:"session"`
}
