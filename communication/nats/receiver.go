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
	callback communication.MessageHandler,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(messageType),
		func(message *nats.Msg) {
			callback(string(message.Data))
		},
	)
	return err
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	callback communication.RequestHandler,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(requestType),
		func(message *nats.Msg) {
			response := callback(string(message.Data))
			receiver.connection.Publish(message.Reply, []byte(response))
		},
	)
	return err
}
