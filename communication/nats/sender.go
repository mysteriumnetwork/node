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
	message communication.Packer,
) error {

	return sender.connection.Publish(
		sender.messageTopic+string(messageType),
		message(),
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request communication.Packer,
	response communication.Unpacker,
) error {

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		request(),
		sender.timeoutRequest,
	)
	if err != nil {
		return err
	}

	response(message.Data)
	return nil
}
