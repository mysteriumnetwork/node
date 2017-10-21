package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"

	log "github.com/cihub/seelog"
)

const SENDER_LOG_PREFIX = "[NATS.Sender] "

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
		log.Warnf("%sFailed to pack message '%s'. %s", SENDER_LOG_PREFIX, messageType, err)
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
		log.Warnf("%sFailed to pack request '%s'. %s", SENDER_LOG_PREFIX, requestType, err)
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
