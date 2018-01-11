package nats_dialog

import (
	"github.com/mysterium/node/communication"
)

// Consume is trying to establish new dialog with Provider
const endpointDialogCreate = communication.RequestEndpoint("dialog-create")

var (
	responseOK              = dialogCreateResponse{200, "OK"}
	responseInvalidIdentity = dialogCreateResponse{400, "Invalid identity"}
)

type dialogCreateRequest struct {
	IdentityId string `json:"identity_id"`
}

type dialogCreateResponse struct {
	Reason        uint   `json:"reason"`
	ReasonMessage string `json:"reasonMessage"`
}
