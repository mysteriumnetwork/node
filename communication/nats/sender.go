package nats

import (
	"github.com/mysterium/node/communication"
	"time"

	"fmt"
	log "github.com/cihub/seelog"
)

const senderLogPrefix = "[NATS.Sender] "

// NewSender constructs new Sender's instance which works thru NATS connection.
// Codec packs/unpacks messages to byte payloads.
// Topic (optional) if need to send messages prefixed topic.
func NewSender(connection Connection, codec communication.Codec, topic string) *senderNats {
	return &senderNats{
		connection:     connection,
		codec:          codec,
		timeoutRequest: 500 * time.Millisecond,
		messageTopic:   topic + ".",
	}
}

type senderNats struct {
	connection     Connection
	codec          communication.Codec
	timeoutRequest time.Duration
	messageTopic   string
}

func (sender *senderNats) Send(producer communication.MessageProducer) error {

	messageTopic := sender.messageTopic + string(producer.GetMessageEndpoint())

	messageData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("failed to encode message '%s'. %s", messageTopic, err)
		return err
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Message '%s' sending: %s", messageTopic, messageData))
	err = sender.connection.Publish(messageTopic, messageData)
	if err != nil {
		err = fmt.Errorf("failed to send message '%s'. %s", messageTopic, err)
		return err
	}

	return nil
}

func (sender *senderNats) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {

	requestTopic := sender.messageTopic + string(producer.GetRequestEndpoint())
	responsePtr = producer.NewResponse()

	requestData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("failed to pack request '%s'. %s", requestTopic, err)
		return
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Request '%s' sending: %s", requestTopic, requestData))
	msg, err := sender.connection.Request(requestTopic, requestData, sender.timeoutRequest)
	if err != nil {
		err = fmt.Errorf("failed to send request '%s'. %s", requestTopic, err)
		return
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Received response for '%s': %s", requestTopic, msg.Data))
	err = sender.codec.Unpack(msg.Data, responsePtr)
	if err != nil {
		err = fmt.Errorf("failed to unpack response '%s'. %s", requestTopic, err)
		log.Error(receiverLogPrefix, err)
		return
	}

	return responsePtr, nil
}
