package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	"fmt"
	log "github.com/cihub/seelog"
)

const RECEIVER_LOG_PREFIX = "[NATS.Receiver] "

type receiverNats struct {
	connection   *nats.Conn
	codec        communication.Codec
	messageTopic string
}

func (receiver *receiverNats) Receive(unpacker communication.MessageUnpacker) error {

	messageType := string(unpacker.GetMessageType())

	messageHandler := func(msg *nats.Msg) {
		messageData := msg.Data
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Message '%s' received: %s", messageType, messageData))

		messagePtr := unpacker.CreateMessage()
		err := receiver.codec.Unpack(msg.Data, messagePtr)
		if err != nil {
			err = fmt.Errorf("Failed to unpack message '%s'. %s", messageType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = unpacker.Handle(messagePtr)
		if err != nil {
			err = fmt.Errorf("Failed to process message '%s'. %s", messageType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}
	}

	_, err := receiver.connection.Subscribe(receiver.messageTopic+messageType, messageHandler)
	if err != nil {
		err = fmt.Errorf("Failed subscribe message '%s'. %s", messageType, err)
		return err
	}

	return nil
}

func (receiver *receiverNats) Respond(unpacker *communication.RequestUnpacker) error {

	requestType := string(unpacker.RequestType)
	messageHandler := func(msg *nats.Msg) {
		requestData := msg.Data
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' received: %s", requestType, requestData))

		err := unpacker.RequestUnpack(requestData)
		if err != nil {
			err = fmt.Errorf("Failed to unpack request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = unpacker.Invoke()
		if err != nil {
			err = fmt.Errorf("Failed to process request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		responseData, err := unpacker.ResponsePack()
		if err != nil {
			err = fmt.Errorf("Failed to pack response '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = receiver.connection.Publish(msg.Reply, responseData)
		if err != nil {
			err = fmt.Errorf("Failed to send response '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' response: %s", requestType, responseData))
	}

	_, err := receiver.connection.Subscribe(receiver.messageTopic+requestType, messageHandler)
	if err != nil {
		err = fmt.Errorf("Failed subscribe request '%s'. %s", requestType, err)
		return err
	}

	return nil
}
