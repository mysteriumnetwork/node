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

func customMessagePacker(message *customMessage) communication.MessagePacker {
	return communication.MessagePacker{
		MessageType: "custom-message",
		Pack: func() ([]byte, error) {
			return json.Marshal(message)
		},
	}
}

func customMessageUnpacker(listener func(*customMessage)) communication.MessageUnpacker {
	var message customMessage

	return communication.MessageUnpacker{
		MessageType: "json-message",
		Unpack: func(data []byte) error {
			return json.Unmarshal(data, &message)
		},
		Invoke: func() error {
			listener(&message)
			return nil
		},
	}
}

func TestMessageCustomSend(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{connection: connection}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("custom-message", func(message *nats.Msg) {
		assert.JSONEq(t, `{"Field": 123}`, string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(
		customMessagePacker(&customMessage{123}),
	)
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

	receiver := &receiverNats{connection: connection}

	messageReceived := make(chan bool)
	err := receiver.Receive(
		customMessageUnpacker(func(message *customMessage) {
			assert.Exactly(t, customMessage{123}, message)
			messageReceived <- true
		}),
	)
	assert.Nil(t, err)

	err = connection.Publish("json-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)
}
