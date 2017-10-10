package nats

import (
	"encoding/json"
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Receiver = &receiverNats{}
}

type rawCallback struct {
	Callback func(message []byte)
}

func (consumer rawCallback) MessageType() communication.MessageType {
	return communication.MessageType("raw-message")
}

func (consumer rawCallback) Consume(messageBody []byte) {
	consumer.Callback(messageBody)
}

func TestReceiverReceiveRaw(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	messageReceived := make(chan bool)
	err := receiver.Receive(rawCallback{func(message []byte) {
		assert.Equal(t, "123", string(message))
		messageReceived <- true
	}})
	assert.Nil(t, err)

	err = connection.Publish("raw-message", []byte("123"))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}

type customMessage struct {
	Field int
}

type customMessageCallback struct {
	Callback func(message customMessage)
}

func (consumer customMessageCallback) MessageType() communication.MessageType {
	return communication.MessageType("json-message")
}

func (consumer customMessageCallback) Consume(messageBody []byte) {
	var message customMessage
	err := json.Unmarshal(messageBody, &message)
	if err != nil {
		panic(err)
	}

	consumer.Callback(message)
}

func TestReceiverReceiveCustomType(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	messageReceived := make(chan bool)
	err := receiver.Receive(&customMessageCallback{func(message customMessage) {
		assert.Equal(t, customMessage{123}, message, "comp2")
		messageReceived <- true
	}})
	assert.Nil(t, err)

	err = connection.Publish("json-message", []byte(`{"Field":123}`))
	assert.Nil(t, err)
}

func TestReceiverRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	requestReceived := make(chan bool)
	err := receiver.Respond(
		communication.RequestType("test"),
		func(request []byte) []byte {
			assert.Equal(t, "REQUEST", string(request))
			requestReceived <- true
			return []byte("RESPONSE")
		},
	)
	assert.Nil(t, err)

	err = connection.PublishRequest("test", "test-reply", []byte("REQUEST"))
	assert.Nil(t, err)

	requestResponded := make(chan bool)
	_, err = connection.Subscribe("test-reply", func(message *nats.Msg) {
		assert.Equal(t, "RESPONSE", string(message.Data))
		requestResponded <- true
	})
	assert.Nil(t, err)

	if err := test.Wait(requestReceived); err != nil {
		t.Fatal("Request not received")
	}

	if err := test.Wait(requestResponded); err != nil {
		t.Fatal("Request not responded")
	}
}
