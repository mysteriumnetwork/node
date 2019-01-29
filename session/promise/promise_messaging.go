package promise

import (
	"github.com/mysteriumnetwork/node/communication"
)

// Request structure represents message from service consumer to send a promise
type Request struct {
	Message Message `json:"promiseMessage"`
}

// Message is the payload we send to the provider
type Message struct {
	Amount     uint64 `json:"amount"`
	SequenceID uint64 `json:"sequenceID"`
	Signature  string `json:"signature"`
}

const endpointPromise = "session-promise"
const messageEndpointPromise = communication.MessageEndpoint(endpointPromise)

// Sender is responsible for sending the promise messages
type Sender struct {
	sender communication.Sender
}

// NewSender returns a new instance of promise sender
func NewSender(sender communication.Sender) *Sender {
	return &Sender{
		sender: sender,
	}
}

// Send send the given promise message
func (ps *Sender) Send(pm Message) error {
	err := ps.sender.Send(&MessageProducer{Message: pm})
	return err
}

// Listener listens for promise messages
type Listener struct {
	MessageConsumer *MessageConsumer
}

// NewListener returns a new instance of promise listener
func NewListener(promiseChan chan Message) *Listener {
	return &Listener{
		MessageConsumer: &MessageConsumer{
			queue: promiseChan,
		},
	}
}

// GetConsumer gets the underlying consumer from the listener
func (pl *Listener) GetConsumer() *MessageConsumer {
	return pl.MessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (pmc *MessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*Request)
	pmc.queue <- request.Message
	return nil
}

// Dialog boilerplate below, please ignore

// MessageConsumer is responsible for consuming the messages
type MessageConsumer struct {
	queue chan Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *MessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

// NewMessage creates struct where request from endpoint will be serialized
func (pmc *MessageConsumer) NewMessage() (requestPtr interface{}) {
	return &Request{}
}

// MessageProducer
type MessageProducer struct {
	Message Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmp *MessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

func (pmp *MessageProducer) Produce() (requestPtr interface{}) {
	return &Request{
		Message: pmp.Message,
	}
}

func (pmp *MessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
