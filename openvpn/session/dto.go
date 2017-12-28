package session

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/session"
)

const SESSION_CREATE = communication.RequestType("session-create")

type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session SessionDto `json:"session"`
}

type SessionDto struct {
	Id     session.SessionId `json:"id"`
	Config json.RawMessage   `json:"config"`
}
