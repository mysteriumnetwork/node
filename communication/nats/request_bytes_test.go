package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type bytesRequestProducer struct {
	Request []byte
}

func (producer *bytesRequestProducer) GetRequestType() communication.RequestType {
	return communication.RequestType("bytes-request")
}

func (producer *bytesRequestProducer) NewResponse() (responsePtr interface{}) {
	var response []byte
	return &response
}

func (producer *bytesRequestProducer) Produce() (requestPtr interface{}) {
	return producer.Request
}

func TestBytesRequest(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		codec:          communication.NewCodecBytes(),
		timeoutRequest: 100 * time.Millisecond,
	}

	requestSent := make(chan bool)
	_, err := connection.Subscribe("bytes-request", func(message *nats.Msg) {
		assert.Equal(t, "REQUEST", string(message.Data))
		connection.Publish(message.Reply, []byte("RESPONSE"))
		requestSent <- true
	})
	assert.Nil(t, err)

	response, err := sender.Request(&bytesRequestProducer{
		[]byte("REQUEST"),
	})
	assert.Nil(t, err)
	assert.Equal(t, []byte("RESPONSE"), *response.(*[]byte))

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

type bytesRequestConsumer struct {
	Callback func(request *[]byte) []byte
}

func (consumer *bytesRequestConsumer) GetRequestType() communication.RequestType {
	return communication.RequestType("bytes-response")
}

func (consumer *bytesRequestConsumer) NewRequest() (requestPtr interface{}) {
	var request []byte
	return &request
}

func (consumer *bytesRequestConsumer) Consume(requestPtr interface{}) (responsePtr interface{}, err error) {
	return consumer.Callback(requestPtr.(*[]byte)), nil
}

func TestBytesRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	requestReceived := make(chan bool)
	err := receiver.Respond(&bytesRequestConsumer{
		func(request *[]byte) []byte {
			assert.Equal(t, []byte("REQUEST"), *request)
			requestReceived <- true
			return []byte("RESPONSE")
		},
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
