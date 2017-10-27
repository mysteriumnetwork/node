package nats

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
)

// Client is trying to establish new dialog with Node
const ENDPOINT_DIALOG_CREATE = communication.RequestType("dialog-create")

type dialogCreateRequest struct {
	IdentityId dto.Identity `json:"identity_id"`
}

func (payload dialogCreateRequest) Pack() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *dialogCreateRequest) Unpack(data []byte) error {
	return json.Unmarshal(data, payload)
}

type dialogCreateResponse struct {
	Accepted bool `json:"accepted"`
}

func (payload dialogCreateResponse) Pack() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *dialogCreateResponse) Unpack(data []byte) error {
	return json.Unmarshal(data, payload)
}
