package nats

import (
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

type dialogCreateUnpacker struct {
	Callback func(request *dialogCreateRequest) (*dialogCreateResponse, error)
}

func (unpacker *dialogCreateUnpacker) GetRequestType() communication.RequestType {
	return ENDPOINT_DIALOG_CREATE
}

func (unpacker *dialogCreateUnpacker) CreateRequest() (requestPtr interface{}) {
	return &dialogCreateRequest{}
}

func (unpacker *dialogCreateUnpacker) Handle(requestPtr interface{}) (responsePtr interface{}, err error) {
	return unpacker.Callback(requestPtr.(*dialogCreateRequest))
}

type dialogCreateRequest struct {
	IdentityId dto.Identity `json:"identity_id"`
}

type dialogCreateResponse struct {
	Accepted bool `json:"accepted"`
}
