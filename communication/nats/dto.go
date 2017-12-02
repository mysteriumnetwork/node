package nats

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
)

// Client is trying to establish new dialog with Node
const ENDPOINT_DIALOG_CREATE = communication.RequestType("dialog-create")

type dialogCreatePacker struct {
	Request *dialogCreateRequest
}

func (packer *dialogCreatePacker) GetRequestType() communication.RequestType {
	return ENDPOINT_DIALOG_CREATE
}

func (packer *dialogCreatePacker) CreateRequest() (requestPtr interface{}) {
	return packer.Request
}

func (packer *dialogCreatePacker) CreateResponse() (responsePtr interface{}) {
	return &dialogCreateResponse{}
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
