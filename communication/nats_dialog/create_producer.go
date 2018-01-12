package nats_dialog

import (
	"github.com/mysterium/node/communication"
)

type dialogCreateProducer struct {
	Request *dialogCreateRequest
}

func (producer *dialogCreateProducer) GetRequestType() communication.RequestType {
	return ENDPOINT_DIALOG_CREATE
}

func (producer *dialogCreateProducer) NewResponse() (responsePtr interface{}) {
	return &dialogCreateResponse{}
}

func (producer *dialogCreateProducer) Produce() (request interface{}) {
	return producer.Request
}
