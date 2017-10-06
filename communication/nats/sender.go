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
	message string,
) error {

	return sender.connection.Publish(
		sender.messageTopic+string(messageType),
		[]byte(message),
	)
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request string,
) (response string, err error) {

	message, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		[]byte(request),
		sender.timeoutRequest,
	)
	if err != nil {
		return
	}

	response = string(message.Data)
	return
}
