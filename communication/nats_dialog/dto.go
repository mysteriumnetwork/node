package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
)

// Consumer is trying to establish new dialog with Provider
const ENDPOINT_DIALOG_CREATE = communication.RequestType("dialog-create")

type dialogCreateRequest struct {
	IdentityId dto.Identity `json:"identity_id"`
}

type dialogCreateResponse struct {
	Accepted bool `json:"accepted"`
}
