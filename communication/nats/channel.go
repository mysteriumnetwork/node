package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type channelNats struct {
	options        nats.Options
	timeoutRequest time.Duration
	myTopic        string

	connection *nats.Conn
}

func (channel *channelNats) Start() (err error) {
	channel.connection, err = channel.options.Connect()
	return err
}

func (channel *channelNats) Stop() error {
	channel.connection.Close()
	return nil
}

func (channel *channelNats) Send(
	messageType communication.MessageType,
	message string,
) error {
	return channel.connection.Publish(string(messageType), []byte(message))
}

func (channel *channelNats) Receive(
	messageType communication.MessageType,
	callback communication.MessageHandler,
) error {

	_, err := channel.connection.Subscribe(
		channel.myTopic+"."+string(messageType),
		func(message *nats.Msg) {
			callback(string(message.Data))
		},
	)
	return err
}

func (channel *channelNats) Request(
	messageType communication.RequestType,
	request string,
) (response string, err error) {
	message, err := channel.connection.Request(string(messageType), []byte(request), channel.timeoutRequest)
	if err != nil {
		return
	}

	response = string(message.Data)
	return
}

func (channel *channelNats) Respond(
	messageType communication.RequestType,
	callback communication.RequestHandler,
) error {

	_, err := channel.connection.Subscribe(
		channel.myTopic+"."+string(messageType),
		func(message *nats.Msg) {
			response := callback(string(message.Data))
			channel.connection.Publish(message.Reply, []byte(response))
		},
	)
	return err
}
