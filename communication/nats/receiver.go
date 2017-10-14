package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
)

type receiverNats struct {
	connection   *nats.Conn
	messageTopic string
}

func (receiver *receiverNats) Receive(
	messageType communication.MessageType,
	handler communication.MessageUnpacker,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(messageType),
		func(message *nats.Msg) {
			handler(message.Data)
		},
	)
	return err
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	consumer communication.RequestConsumer,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(requestType),
		func(message *nats.Msg) {
			requestData := message.Data
			responseData := consumer.ConsumeRequest(requestData)
			receiver.connection.Publish(message.Reply, responseData)
		},
	)
	return err
}
