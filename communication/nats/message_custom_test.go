package nats

import (
	"encoding/json"
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

func customMessageUnpacker(listener func(customMessage)) *communication.MessageUnpacker {
	var message customMessage

	return &communication.MessageUnpacker{
		MessageType: "json-message",
		Unpack: func(data []byte) error {
			return json.Unmarshal(data, &message)
		},
		Invoke: func() error {
			listener(message)
			return nil
		},
	}
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

func TestMessageCustomReceive(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
	}

	messageReceived := make(chan bool)
	err := receiver.Receive(
		customMessageUnpacker(func(message customMessage) {
			assert.Exactly(t, customMessage{123}, message)
			messageReceived <- true
		}),
	)
	assert.Nil(t, err)

	err = connection.Publish("json-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}
