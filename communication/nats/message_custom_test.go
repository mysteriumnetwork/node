package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
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
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("custom-message", func(message *nats.Msg) {
		assert.JSONEq(t, `{"Field": 123}`, string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(&customMessageProducer{&customMessage{123}})
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

func TestMessageCustomSendNull(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("custom-message", func(message *nats.Msg) {
		assert.JSONEq(t, `null`, string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(&customMessageProducer{})
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

type customMessageConsumer struct {
	Callback func(message *customMessage)
}

func (consumer *customMessageConsumer) GetMessageType() communication.MessageType {
	return communication.MessageType("custom-message")
}

func (consumer *customMessageConsumer) NewMessage() (messagePtr interface{}) {
	return &customMessage{}
}

func (consumer *customMessageConsumer) Consume(messagePtr interface{}) error {
	consumer.Callback(messagePtr.(*customMessage))
	return nil
}

func TestMessageCustomReceive(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	messageReceived := make(chan bool)
	err := receiver.Receive(&customMessageConsumer{func(message *customMessage) {
		assert.Exactly(t, customMessage{123}, *message)
		messageReceived <- true
	}})
	assert.Nil(t, err)

	err = connection.Publish("custom-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}
