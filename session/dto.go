package session

import (
	"github.com/mysterium/node/communication"
)

const endpointSessionCreate = communication.RequestEndpoint("session-create")

type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session SessionDto `json:"session"`
}

type SessionDto struct {
	ID     SessionID `json:"id"`
	Config string    `json:"config"`
}
