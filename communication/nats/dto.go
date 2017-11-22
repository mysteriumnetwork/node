package nats

import (
	"encoding/json"
	"fmt"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
)

// Client is trying to establish new dialog with Node
const ENDPOINT_DIALOG_CREATE = communication.RequestType("dialog-create")

func requestDialogCreate(sender communication.Sender, request dialogCreateRequest) (response *dialogCreateResponse, err error) {
	err = sender.Request(&communication.RequestPacker{
		RequestType: ENDPOINT_DIALOG_CREATE,
		RequestPack: func() ([]byte, error) {
			return json.Marshal(request)
		},
		ResponseUnpack: func(responseData []byte) error {
			return json.Unmarshal(responseData, response)
		},
	})
	if !response.Accepted {
		err = fmt.Errorf("Dialog creation rejected: %s", response)
	}

	return response, err
}

func respondDialogCreate(receiver communication.Receiver, callback func(request *dialogCreateRequest) *dialogCreateResponse) error {
	var request *dialogCreateRequest
	var response *dialogCreateResponse

	return receiver.Respond(&communication.RequestUnpacker{
		RequestType: ENDPOINT_DIALOG_CREATE,
		RequestUnpack: func(requestData []byte) error {
			return json.Unmarshal(requestData, &request)
		},
		ResponsePack: func() ([]byte, error) {
			return json.Marshal(response)
		},
		Invoke: func() error {
			response = callback(request)
			return nil
		},
	})
}

type dialogCreateRequest struct {
	IdentityId dto.Identity `json:"identity_id"`
}

type dialogCreateResponse struct {
	Accepted bool `json:"accepted"`
}
