package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBytesRequest(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		timeoutRequest: 100 * time.Millisecond,
	}

	requestSent := make(chan bool)
	_, err := connection.Subscribe("bytes-request", func(message *nats.Msg) {
		assert.Equal(t, "REQUEST", string(message.Data))
		connection.Publish(message.Reply, []byte("RESPONSE"))
		requestSent <- true
	})
	assert.Nil(t, err)

	response := communication.BytesResponse{}
	err = sender.Request(
		communication.RequestType("bytes-request"),
		communication.BytesProduce{[]byte("REQUEST")},
		&response,
	)
	assert.Nil(t, err)
	assert.Equal(t, "RESPONSE", string(response.Response))

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

func TestBytesRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	requestReceived := make(chan bool)
	err := receiver.Respond(
		communication.RequestType("bytes-response"),
		func(request []byte) []byte {
			assert.Equal(t, "REQUEST", string(request))
			requestReceived <- true
			return []byte("RESPONSE")
		},
	)
	assert.Nil(t, err)

	err = connection.PublishRequest("bytes-response", "bytes-reply", []byte("REQUEST"))
	assert.Nil(t, err)

	requestResponded := make(chan bool)
	_, err = connection.Subscribe("bytes-reply", func(message *nats.Msg) {
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
