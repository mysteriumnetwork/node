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

	messageData, err := message.Pack()
	if err != nil {
		return err
	}

	return sender.connection.Publish(
		sender.messageTopic+string(messageType),
		messageData,
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request communication.Packer,
	response communication.Unpacker,
) error {

	requestData, err := request.Pack()
	if err != nil {
		return err
	}

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		requestData,
		sender.timeoutRequest,
	)
	if err != nil {
		return err
	}

	err = response.Unpack(message.Data)
	return err
}
