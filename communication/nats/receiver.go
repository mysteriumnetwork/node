package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	"fmt"
	log "github.com/cihub/seelog"
)

const receiverLogPrefix = "[NATS.Receiver] "

// NewReceiver constructs new Receiver's instance which works thru NATS connection.
// Codec packs/unpacks messages to byte payloads.
// Topic (optional) if need to send messages prefixed topic.
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

	messageTopic := receiver.messageTopic + string(consumer.GetMessageEndpoint())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(receiverLogPrefix, fmt.Sprintf("Message '%s' received: %s", messageTopic, msg.Data))
		messagePtr := consumer.NewMessage()
		err := receiver.codec.Unpack(msg.Data, messagePtr)
		if err != nil {
			err = fmt.Errorf("failed to unpack message '%s'. %s", messageTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		err = consumer.Consume(messagePtr)
		if err != nil {
			err = fmt.Errorf("failed to process message '%s'. %s", messageTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}
	}

	_, err := receiver.connection.Subscribe(messageTopic, messageHandler)
	if err != nil {
		err = fmt.Errorf("failed subscribe message '%s'. %s", messageTopic, err)
		return err
	}

	return nil
}

func (receiver *receiverNats) Respond(consumer communication.RequestConsumer) error {

	requestTopic := receiver.messageTopic + string(consumer.GetRequestEndpoint())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(receiverLogPrefix, fmt.Sprintf("Request '%s' received: %s", requestTopic, msg.Data))
		requestPtr := consumer.NewRequest()
		err := receiver.codec.Unpack(msg.Data, requestPtr)
		if err != nil {
			err = fmt.Errorf("failed to unpack request '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		response, err := consumer.Consume(requestPtr)
		if err != nil {
			err = fmt.Errorf("failed to process request '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		responseData, err := receiver.codec.Pack(response)
		if err != nil {
			err = fmt.Errorf("failed to pack response '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		log.Debug(receiverLogPrefix, fmt.Sprintf("Request '%s' response: %s", requestTopic, responseData))
		err = receiver.connection.Publish(msg.Reply, responseData)
		if err != nil {
			err = fmt.Errorf("failed to send response '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}
	}

	_, err := receiver.connection.Subscribe(requestTopic, messageHandler)
	if err != nil {
		err = fmt.Errorf("failed subscribe request '%s'. %s", requestTopic, err)
		return err
	}

	return nil
}
