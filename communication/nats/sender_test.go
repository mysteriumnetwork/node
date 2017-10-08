package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSenderInterface(t *testing.T) {
	var _ communication.Sender = &senderNats{}
}

func TestSenderSend(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{connection: connection}

	messageSent := make(chan bool)
	_, err := connection.Subscribe("test", func(message *nats.Msg) {
		assert.Equal(t, "123", string(message.Data))
		messageSent <- true
	})
	assert.Nil(t, err)

	err = sender.Send(
		communication.MessageType("test"),
		[]byte("123"),
	)
	assert.Nil(t, err)

	if err := test.Wait(messageSent); err != nil {
		t.Fatal("Message not sent")
	}
}

func TestSenderRequest(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		timeoutRequest: 100 * time.Millisecond,
	}

	requestSent := make(chan bool)
	_, err := connection.Subscribe("test", func(message *nats.Msg) {
		assert.Equal(t, "REQUEST", string(message.Data))
		connection.Publish(message.Reply, []byte("RESPONSE"))
		requestSent <- true
	})
	assert.Nil(t, err)

	response, err := sender.Request(
		communication.RequestType("test"),
		[]byte("REQUEST"),
	)
	assert.Nil(t, err)
	assert.Equal(t, "RESPONSE", string(response))

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}
