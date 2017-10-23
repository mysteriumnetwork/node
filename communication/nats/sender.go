package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"time"

	"fmt"
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
		err = fmt.Errorf("Failed to pack message '%s'. %s", messageType, err)
		return err
	}

	log.Debug(SENDER_LOG_PREFIX, fmt.Sprintf("Message '%s' sending: %s", messageType, messageData))
	err = sender.connection.Publish(
		sender.messageTopic+string(messageType),
		messageData,
	)
	if err != nil {
		err = fmt.Errorf("Failed to send message '%s'. %s", messageType, err)
		return err
	}

	return nil
}

func (sender *senderNats) Request(
	requestType communication.RequestType,
	request communication.Packer,
	response communication.Unpacker,
) error {

	requestData, err := request.Pack()
	if err != nil {
		err = fmt.Errorf("Failed to pack request '%s'. %s", requestType, err)
		return err
	}

	log.Debug(SENDER_LOG_PREFIX, fmt.Sprintf("Request '%s' sending: %s", requestType, requestData))
	msg, err := sender.connection.Request(
		sender.messageTopic+string(requestType),
		requestData,
		sender.timeoutRequest,
	)
	if err != nil {
		err = fmt.Errorf("Failed to send request '%s'. %s", requestType, err)
		return err
	}

	responseData := msg.Data
	log.Debug(SENDER_LOG_PREFIX, fmt.Sprintf("Received response for '%s': %s", requestType, responseData))

	err = response.Unpack(responseData)
	if err != nil {
		err = fmt.Errorf("Failed to unpack response '%s'. %s", requestType, err)
		return err
	}

	return nil
}
