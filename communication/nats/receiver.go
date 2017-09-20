package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
)

type receiverNats struct {
	connection *nats.Conn
	myTopic    string
}

func (server *receiverNats) Receive(
	messageType communication.MessageType,
	callback communication.MessageHandler,
) error {

	_, err := server.connection.Subscribe(
		server.myTopic+"."+string(messageType),
		func(message *nats.Msg) {
			callback(string(message.Data))
		},
	)
	return err
}

func (server *receiverNats) Respond(
	messageType communication.RequestType,
	callback communication.RequestHandler,
) error {

	_, err := server.connection.Subscribe(
		server.myTopic+"."+string(messageType),
		func(message *nats.Msg) {
			response := callback(string(message.Data))
			server.connection.Publish(message.Reply, []byte(response))
		},
	)
	return err
}
