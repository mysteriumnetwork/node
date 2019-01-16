package balance

import (
	"github.com/mysteriumnetwork/node/communication"
)

// DestroyRequest structure represents message from service consumer to destroy session for given session id
type BalanceRequest struct {
	BalanceMessage BalanceMessage `json:"balanceMessage"`
}

// DestroyResponse structure represents service provider response to given session request from consumer
type BalanceResponse struct{}

type BalanceMessage struct {
	Balance    uint64 `json:"balance"`
	SequenceID uint32 `json:"sequenceID"`
}

const endpointBalance = "session-balance"
const requestEndpointBalance = communication.RequestEndpoint(endpointBalance)
const messageEndpointBalance = communication.MessageEndpoint(endpointBalance)

type BalanceSender struct {
	sender communication.Sender
}

func NewBalanceSender(sender communication.Sender) *BalanceSender {
	return &BalanceSender{
		sender: sender,
	}
}

func (bs *BalanceSender) Send(bm BalanceMessage) error {
	_, err := bs.sender.Request(&balanceMessageProducer{BalanceMessage: bm})
	return err
}

type BalanceListener struct {
	balanceMessageConsumer *balanceMessageConsumer
}

func NewBalanceListener() *BalanceListener {
	return &BalanceListener{
		balanceMessageConsumer: &balanceMessageConsumer{
			queue: make(chan BalanceMessage, 1),
		},
	}
}

func (bl *BalanceListener) Listen() <-chan BalanceMessage {
	return bl.balanceMessageConsumer.queue
}

func (bl *BalanceListener) GetConsumer() *balanceMessageConsumer {
	return bl.balanceMessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (bmc *balanceMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*BalanceRequest)
	bmc.queue <- request.BalanceMessage
	return nil
}

// Dialog boilerplate below, please ignore

type balanceMessageConsumer struct {
	queue chan BalanceMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (bmc *balanceMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointBalance
}

// NewRequest creates struct where request from endpoint will be serialized
func (bmc *balanceMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &BalanceRequest{}
}

// balanceMessageProducer
type balanceMessageProducer struct {
	BalanceMessage BalanceMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (bmp *balanceMessageProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return requestEndpointBalance
}

func (bmp *balanceMessageProducer) Produce() (requestPtr interface{}) {
	return &BalanceRequest{
		BalanceMessage: BalanceMessage{},
	}
}

func (bmp *balanceMessageProducer) NewResponse() (responsePtr interface{}) {
	return &BalanceResponse{}
}
