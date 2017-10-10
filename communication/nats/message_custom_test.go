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

type customMessageCallback struct {
	Callback func(message customMessage)
}

func (consumer customMessageCallback) ConsumeMessage(messageBody []byte) {
	var message customMessage
	err := json.Unmarshal(messageBody, &message)
	if err != nil {
		panic(err)
	}

	consumer.Callback(message)
}

type customMessageProduce struct {
	Message customMessage
}

func (producer customMessageProduce) ProduceMessage() []byte {
	messageBody, err := json.Marshal(producer.Message)
	if err != nil {
		panic(err)
	}
	return messageBody
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
		customMessageProduce{
			customMessage{123},
		},
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
		communication.MessageType("json-message"),
		customMessageCallback{func(message customMessage) {
			assert.Equal(t, customMessage{123}, message)
			messageReceived <- true
		}},
	)
	assert.Nil(t, err)

	err = connection.Publish("json-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)
}
