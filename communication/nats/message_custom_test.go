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

type customMessagePacker struct {
	Message *customMessage
}

func (packer *customMessagePacker) GetMessageType() communication.MessageType {
	return communication.MessageType("custom-message")
}

func (packer *customMessagePacker) CreateMessage() (messagePtr interface{}) {
	return packer.Message
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

	err = sender.Send(&customMessagePacker{&customMessage{123}})
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

	err = sender.Send(&customMessagePacker{})
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

type customMessageUnpacker struct {
	Callback func(message *customMessage)
}

func (unpacker *customMessageUnpacker) GetMessageType() communication.MessageType {
	return communication.MessageType("custom-message")
}

func (unpacker *customMessageUnpacker) CreateMessage() (messagePtr interface{}) {
	return &customMessage{}
}

func (unpacker *customMessageUnpacker) Handle(messagePtr interface{}) error {
	unpacker.Callback(messagePtr.(*customMessage))
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
	err := receiver.Receive(&customMessageUnpacker{func(message *customMessage) {
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
