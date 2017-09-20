package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"
)

type senderNats struct {
	connection     *nats.Conn
	timeoutRequest time.Duration
}

func (client *senderNats) Send(
	messageType communication.MessageType,
	message string,
) error {
	return client.connection.Publish(string(messageType), []byte(message))
}

func (client *senderNats) Request(
	messageType communication.RequestType,
	request string,
) (response string, err error) {
	message, err := client.connection.Request(string(messageType), []byte(request), client.timeoutRequest)
	if err != nil {
		return
	}

	response = string(message.Data)
	return
}
