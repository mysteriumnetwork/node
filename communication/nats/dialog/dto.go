package dialog

import (
	"github.com/mysterium/node/communication"
)

// Consume is trying to establish new dialog with Provider
const endpointDialogCreate = communication.RequestEndpoint("dialog-create")

var (
	responseOK              = dialogCreateResponse{200, "OK"}
	responseInvalidIdentity = dialogCreateResponse{400, "Invalid identity"}
	responseInternalError   = dialogCreateResponse{500, "Failed to create dialog"}
)

type dialogCreateRequest struct {
	PeerID string `json:"peer_id"`
}

type dialogCreateResponse struct {
	Reason        uint   `json:"reason"`
	ReasonMessage string `json:"reasonMessage"`
}
