package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
)

type receiverNats struct {
	connection   *nats.Conn
	messageTopic string
}

func (receiver *receiverNats) Receive(handler communication.MessageHandler) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(handler.Type()),
		func(message *nats.Msg) {
			handler.Deliver(message.Data)
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
			response := callback(message.Data)
			receiver.connection.Publish(message.Reply, []byte(response))
		},
	)
	return err
}
