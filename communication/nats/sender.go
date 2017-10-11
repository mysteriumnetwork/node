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

func (sender *senderNats) Send(
	messageType communication.MessageType,
	producer communication.MessageProducer,
) error {

	return sender.connection.Publish(
		sender.messageTopic+string(messageType),
		producer.ProduceMessage(),
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request communication.MessageProducer,
	response communication.MessageConsumer,
) error {

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		request.ProduceMessage(),
		sender.timeoutRequest,
	)
	if err != nil {
		return err
	}

	response.ConsumeMessage(message.Data)
	return nil
}
