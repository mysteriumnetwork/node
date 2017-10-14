package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageBytesSend(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{connection: connection}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("bytes-message", func(message *nats.Msg) {
		assert.Equal(t, "123", string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(
		communication.MessageType("bytes-message"),
		communication.BytesPacker([]byte("123")),
	)
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

func TestMessageBytesSendReceive(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	messageReceived := make(chan bool)
	err := receiver.Receive(
		communication.MessageType("bytes-message"),
		func(message []byte) {
			assert.Equal(t, "123", string(message))
			messageReceived <- true
		},
	)
	assert.Nil(t, err)

	err = connection.Publish("bytes-message", []byte("123"))
	assert.Nil(t, err)

	if err := test.Wait(messageReceived); err != nil {
		t.Fatal("Message not received")
	}
}
