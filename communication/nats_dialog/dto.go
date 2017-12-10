package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
)

// Consumer is trying to establish new dialog with Provider
const ENDPOINT_DIALOG_CREATE = communication.RequestType("dialog-create")

type dialogCreateProducer struct {
	Request *dialogCreateRequest
}

func (producer *dialogCreateProducer) GetRequestType() communication.RequestType {
	return ENDPOINT_DIALOG_CREATE
}

func (producer *dialogCreateProducer) NewResponse() (responsePtr interface{}) {
	return &dialogCreateResponse{}
}

func (producer *dialogCreateProducer) Produce() (requestPtr interface{}) {
	return producer.Request
}

type dialogCreateHandler struct {
	Callback func(request *dialogCreateRequest) (*dialogCreateResponse, error)
}

func (handler *dialogCreateHandler) GetRequestType() communication.RequestType {
	return ENDPOINT_DIALOG_CREATE
}

func (handler *dialogCreateHandler) NewRequest() (requestPtr interface{}) {
	return &dialogCreateRequest{}
}

func (handler *dialogCreateHandler) Handle(requestPtr interface{}) (responsePtr interface{}, err error) {
	return handler.Callback(requestPtr.(*dialogCreateRequest))
}

type dialogCreateRequest struct {
	IdentityId dto.Identity `json:"identity_id"`
}

type dialogCreateResponse struct {
	Accepted bool `json:"accepted"`
}
