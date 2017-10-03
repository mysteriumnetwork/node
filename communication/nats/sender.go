package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type senderNats struct {
	connection     *nats.Conn
	receiverTopic  string
	timeoutRequest time.Duration
}

func (sender *senderNats) Send(
	messageType communication.MessageType,
	message string,
) error {
	return sender.connection.Publish(
		string(messageType),
		[]byte(message),
	)
}

func (sender *senderNats) Request(
	messageType communication.RequestType,
	request string,
) (response string, err error) {
	message, err := sender.connection.Request(string(messageType), []byte(request), sender.timeoutRequest)
	if err != nil {
		return
	}

	response = string(message.Data)
	return
}
