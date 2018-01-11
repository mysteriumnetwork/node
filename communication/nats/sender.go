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

	messageType := string(producer.GetMessageEndpoint())

	messageData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("Failed to encode message '%s'. %s", messageType, err)
		return err
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Message '%s' sending: %s", messageType, messageData))
	err = sender.connection.Publish(
		sender.messageTopic+messageType,
		messageData,
	)
	if err != nil {
		err = fmt.Errorf("Failed to send message '%s'. %s", messageType, err)
		return err
	}

	return nil
}

func (sender *senderNats) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {

	requestType := string(producer.GetRequestEndpoint())
	responsePtr = producer.NewResponse()

	requestData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("Failed to pack request '%s'. %s", requestType, err)
		return
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Request '%s' sending: %s", requestType, requestData))
	msg, err := sender.connection.Request(
		sender.messageTopic+requestType,
		requestData,
		sender.timeoutRequest,
	)
	if err != nil {
		err = fmt.Errorf("Failed to send request '%s'. %s", requestType, err)
		return
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Received response for '%s': %s", requestType, msg.Data))
	err = sender.codec.Unpack(msg.Data, responsePtr)
	if err != nil {
		err = fmt.Errorf("Failed to unpack response '%s'. %s", requestType, err)
		log.Error(receiverLogPrefix, err)
		return
	}

	return responsePtr, nil
}
