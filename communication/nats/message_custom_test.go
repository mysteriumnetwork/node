package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

type customMessage struct {
	Field int
}

type customMessageProducer struct {
	Message *customMessage
}

func (producer *customMessageProducer) GetMessageType() communication.MessageType {
	return communication.MessageType("custom-message")
}

func (producer *customMessageProducer) Produce() (messagePtr interface{}) {
	return producer.Message
}

func TestMessageCustomSend(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Stop()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	err := sender.Send(&customMessageProducer{&customMessage{123}})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"Field": 123}`, string(connection.GetLastMessage()))
}

func TestMessageCustomSendNull(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Stop()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	err := sender.Send(&customMessageProducer{})
	assert.NoError(t, err)
	assert.JSONEq(t, `null`, string(connection.GetLastMessage()))
}

type customMessageConsumer struct {
	messageReceived chan interface{}
}

func (consumer *customMessageConsumer) GetMessageType() communication.MessageType {
	return communication.MessageType("custom-message")
}

func (consumer *customMessageConsumer) NewMessage() (messagePtr interface{}) {
	return &customMessage{}
}

func (consumer *customMessageConsumer) Consume(messagePtr interface{}) error {
	consumer.messageReceived <- messagePtr
	return nil
}

func TestMessageCustomReceive(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Stop()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	consumer := &customMessageConsumer{messageReceived: make(chan interface{})}
	err := receiver.Receive(consumer)
	assert.NoError(t, err)

	connection.Publish("custom-message", []byte(`{"Field":123}`))
	message, err := connection.MessageWait(consumer.messageReceived)
	assert.NoError(t, err)
	assert.Exactly(t, customMessage{123}, *message.(*customMessage))
}
