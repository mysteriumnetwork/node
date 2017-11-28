package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func requestBytes(sender communication.Sender, request []byte) (response []byte, err error) {
	err = sender.Request(&communication.RequestPacker{
		RequestType: "bytes-request",
		RequestPack: func() ([]byte, error) {
			return request, nil
		},
		ResponseUnpack: func(responseData []byte) error {
			response = responseData
			return nil
		},
	})
	return response, err
}

func TestBytesRequest(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		codec:          communication.NewCodecJSON(),
		timeoutRequest: 100 * time.Millisecond,
	}

	requestSent := make(chan bool)
	_, err := connection.Subscribe("bytes-request", func(message *nats.Msg) {
		assert.Equal(t, "REQUEST", string(message.Data))
		connection.Publish(message.Reply, []byte("RESPONSE"))
		requestSent <- true
	})
	assert.Nil(t, err)

	response, err := requestBytes(sender, []byte("REQUEST"))
	assert.Nil(t, err)
	assert.Equal(t, "RESPONSE", string(response))

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

func respondBytes(receiver communication.Receiver, callback func([]byte) []byte) error {
	var request []byte
	var response []byte

	return receiver.Respond(&communication.RequestUnpacker{
		RequestType: "bytes-response",
		RequestUnpack: func(requestData []byte) error {
			request = requestData
			return nil
		},
		ResponsePack: func() ([]byte, error) {
			return response, nil
		},
		Invoke: func() error {
			response = callback(request)
			return nil
		},
	})
}

func TestBytesRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
	}

	requestReceived := make(chan bool)
	err := respondBytes(receiver, func(request []byte) []byte {
		assert.Equal(t, "REQUEST", string(request))
		requestReceived <- true
		return []byte("RESPONSE")
	})
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
