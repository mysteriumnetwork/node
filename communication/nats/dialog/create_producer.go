package dialog

import (
	"github.com/mysterium/node/communication"
)

type dialogCreateProducer struct {
	Request *dialogCreateRequest
}

func (producer *dialogCreateProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointDialogCreate
}

func (producer *dialogCreateProducer) NewResponse() (responsePtr interface{}) {
	return &dialogCreateResponse{}
}

func (producer *dialogCreateProducer) Produce() (requestPtr interface{}) {
	return producer.Request
}
