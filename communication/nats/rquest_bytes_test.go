package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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
