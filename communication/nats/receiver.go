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
	listener communication.MessageListener,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(messageType),
		func(message *nats.Msg) {
			listener(message.Data)
		},
	)
	return err
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	handler communication.RequestHandler,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(requestType),
		func(message *nats.Msg) {
			request := message.Data
			response := handler(request)
			receiver.connection.Publish(message.Reply, response)
		},
	)
	return err
}
