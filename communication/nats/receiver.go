package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	log "github.com/cihub/seelog"
)

const RECEIVER_LOG_PREFIX = "[NATS.Receiver] "

type receiverNats struct {
	connection   *nats.Conn
	messageTopic string
}

func (receiver *receiverNats) Receive(
	messageType communication.MessageType,
	listener communication.MessageListener,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(messageType),
		func(msg *nats.Msg) {
			err := listener.Message.Unpack(msg.Data)
			if err != nil {
				log.Warnf("%sFailed to unpack message '%s'. %s", RECEIVER_LOG_PREFIX, messageType, err)
				return
			}

			listener.Invoke()
		},
	)
	return err
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	handler communication.RequestHandler,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(requestType),
		func(message *nats.Msg) {
			request := message.Data
			response := handler(request)
			receiver.connection.Publish(message.Reply, response)
		},
	)
	return err
}
