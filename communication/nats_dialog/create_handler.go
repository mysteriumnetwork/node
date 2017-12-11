package nats_dialog

import (
	"github.com/mysterium/node/communication"
)

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
