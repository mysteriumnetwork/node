package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type senderNats struct {
	connection     *nats.Conn
	timeoutRequest time.Duration
	messageTopic   string
}

func (sender *senderNats) Send(producer communication.MessageProducer) error {

	return sender.connection.Publish(
		sender.messageTopic+string(producer.MessageType()),
		producer.Produce(),
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request []byte,
) (response []byte, err error) {

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		request,
		sender.timeoutRequest,
	)
	if err != nil {
		return
	}

	response = message.Data
	return
}
