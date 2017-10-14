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
	messagePacker communication.Packer,
) error {

	return sender.connection.Publish(
		sender.messageTopic+string(messageType),
		messagePacker(),
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	requestPacker communication.Packer,
	responseUnpacker communication.Unpacker,
) error {

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		requestPacker(),
		sender.timeoutRequest,
	)
	if err != nil {
		return err
	}

	responseUnpacker(message.Data)
	return nil
}
