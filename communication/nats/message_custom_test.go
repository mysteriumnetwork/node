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

func (payload customMessage) Pack() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *customMessage) Unpack(data []byte) error {
	return json.Unmarshal(data, payload)
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
		communication.MessageType("custom-message"),
		customMessage{123},
	)
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

func customMessageListener(listener func(*customMessage)) communication.MessageListener {
	return func(messageData []byte) {
		var message customMessage
		err := message.Unpack(messageData)
		if err != nil {
			panic(err)
		}

		listener(&message)
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
		communication.MessageType("json-message"),
		customMessageListener(func(message *customMessage) {
			assert.Equal(t, customMessage{123}, message)
			messageReceived <- true
		}),
	)
	assert.Nil(t, err)

	err = connection.Publish("json-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)
}
