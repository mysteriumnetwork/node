package dialog

import (
	"github.com/mysterium/node/communication"
)

type dialogCreateConsumer struct {
	Callback func(request dialogCreateRequest) (dialogCreateResponse, error)
}

func (consumer *dialogCreateConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointDialogCreate
}

func (consumer *dialogCreateConsumer) NewRequest() (requestPtr interface{}) {
	return &dialogCreateRequest{}
}

func (consumer *dialogCreateConsumer) Consume(request interface{}) (response interface{}, err error) {
	return consumer.Callback(request.(dialogCreateRequest))
}
