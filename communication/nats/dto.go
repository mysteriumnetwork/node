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
