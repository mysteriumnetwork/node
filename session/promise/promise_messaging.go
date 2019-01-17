package promise

import (
	"github.com/mysteriumnetwork/node/communication"
)

// DestroyRequest structure represents message from service consumer to destroy session for given session id
type PromiseRequest struct {
	PromiseMessage PromiseMessage `json:"balanceMessage"`
}

type PromiseMessage struct {
	Amount     uint64 `json:"amount"`
	SequenceID uint64 `json:"sequenceID"`
	Signature  string `json:"signature"`
}

const endpointPromise = "session-promise"
const messageEndpointPromise = communication.MessageEndpoint(endpointPromise)

type PromiseSender struct {
	sender communication.Sender
}

func NewPromiseSender(sender communication.Sender) *PromiseSender {
	return &PromiseSender{
		sender: sender,
	}
}

func (ps *PromiseSender) Send(pm PromiseMessage) error {
	err := ps.sender.Send(&promiseMessageProducer{PromiseMessage: pm})
	return err
}

type PromiseListener struct {
	promiseMessageConsumer *promiseMessageConsumer
}

func NewPromiseListener() *PromiseListener {
	return &PromiseListener{
		promiseMessageConsumer: &promiseMessageConsumer{
			queue: make(chan PromiseMessage, 1),
		},
	}
}

func (pl *PromiseListener) Listen() <-chan PromiseMessage {
	return pl.promiseMessageConsumer.queue
}

func (pl *PromiseListener) GetConsumer() *promiseMessageConsumer {
	return pl.promiseMessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (pmc *promiseMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*PromiseRequest)
	pmc.queue <- request.PromiseMessage
	return nil
}

// Dialog boilerplate below, please ignore

type promiseMessageConsumer struct {
	queue chan PromiseMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *promiseMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

// NewRequest creates struct where request from endpoint will be serialized
func (pmc *promiseMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &PromiseRequest{}
}

// promiseMessageProducer
type promiseMessageProducer struct {
	PromiseMessage PromiseMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmp *promiseMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

func (pmp *promiseMessageProducer) Produce() (requestPtr interface{}) {
	return &PromiseRequest{
		PromiseMessage: PromiseMessage{},
	}
}

func (pmp *promiseMessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
