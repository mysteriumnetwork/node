package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	"fmt"
	log "github.com/cihub/seelog"
)

const RECEIVER_LOG_PREFIX = "[NATS.Receiver] "

func newReceiver(connection *nats.Conn, messageTopic string, codec communication.Codec) *receiverNats {
	if codec == nil {
		codec = communication.NewCodecJSON()
	}
	if messageTopic != "" {
		messageTopic = messageTopic + "."
	}

	return &receiverNats{
		connection:   connection,
		codec:        codec,
		messageTopic: messageTopic,
	}
}

type receiverNats struct {
	connection   *nats.Conn
	codec        communication.Codec
	messageTopic string
}

func (receiver *receiverNats) Receive(handler communication.MessageHandler) error {

	messageType := string(handler.GetMessageType())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Message '%s' received: %s", messageType, msg.Data))
		messagePtr := handler.NewMessage()
		err := receiver.codec.Unpack(msg.Data, messagePtr)
		if err != nil {
			err = fmt.Errorf("Failed to unpack message '%s'. %s", messageType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = handler.Handle(messagePtr)
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

func (receiver *receiverNats) Respond(handler communication.RequestHandler) error {

	requestType := string(handler.GetRequestType())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' received: %s", requestType, msg.Data))
		requestPtr := handler.NewRequest()
		err := receiver.codec.Unpack(msg.Data, requestPtr)
		if err != nil {
			err = fmt.Errorf("Failed to unpack request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		response, err := handler.Handle(requestPtr)
		if err != nil {
			err = fmt.Errorf("Failed to process request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		responseData, err := receiver.codec.Pack(response)
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
