package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	"fmt"
	log "github.com/cihub/seelog"
)

const RECEIVER_LOG_PREFIX = "[NATS.Receiver] "

func NewReceiver(connection Connection, codec communication.Codec, topic string) *receiverNats {
	return &receiverNats{
		connection:   connection,
		codec:        codec,
		messageTopic: topic + ".",
	}
}

type receiverNats struct {
	connection   Connection
	codec        communication.Codec
	messageTopic string
}

func (receiver *receiverNats) Receive(consumer communication.MessageConsumer) error {

	messageType := string(consumer.GetMessageType())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Message '%s' received: %s", messageType, msg.Data))
		messagePtr := consumer.NewMessage()
		err := receiver.codec.Unpack(msg.Data, messagePtr)
		if err != nil {
			err = fmt.Errorf("Failed to unpack message '%s'. %s", messageType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = consumer.Consume(messagePtr)
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

func (receiver *receiverNats) Respond(consumer communication.RequestConsumer) error {

	requestType := string(consumer.GetRequestEndpoint())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' received: %s", requestType, msg.Data))
		requestPtr := consumer.NewRequest()
		err := receiver.codec.Unpack(msg.Data, requestPtr)
		if err != nil {
			err = fmt.Errorf("Failed to unpack request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		response, err := consumer.Consume(requestPtr)
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

		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' response: %s", requestType, responseData))
		err = receiver.connection.Publish(msg.Reply, responseData)
		if err != nil {
			err = fmt.Errorf("Failed to send response '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}
	}

	_, err := receiver.connection.Subscribe(receiver.messageTopic+requestType, messageHandler)
	if err != nil {
		err = fmt.Errorf("Failed subscribe request '%s'. %s", requestType, err)
		return err
	}

	return nil
}
