package session

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
)

const endpointSessionCreate = communication.RequestEndpoint("session-create")

// SessionCreateRequest structure represents message from service consumer to initiate session for given proposal id
type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

// SessionCreateResponse structure represents service provider response to given session request from consumer
type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session SessionDto `json:"session"`
}

// SessionDto structure represents session information data within session creation response (session id and configuration options for underlaying service type)
type SessionDto struct {
	ID     SessionID       `json:"id"`
	Config json.RawMessage `json:"config"`
}

// ServiceConfiguration defines service configuration from underlying transport mechanism to be passed to remote party
// should be serializable to json format
type ServiceConfiguration interface{}
