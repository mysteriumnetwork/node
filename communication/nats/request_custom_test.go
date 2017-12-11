package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type customRequest struct {
	FieldIn string
}

type customResponse struct {
	FieldOut string
}

type customRequestProducer struct {
	Request *customRequest
}

func (producer *customRequestProducer) GetRequestType() communication.RequestType {
	return communication.RequestType("custom-request")
}

func (producer *customRequestProducer) NewResponse() (responsePtr interface{}) {
	return &customResponse{}
}

func (producer *customRequestProducer) Produce() (requestPtr interface{}) {
	return producer.Request
}

func TestCustomRequest(t *testing.T) {
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
	_, err := connection.Subscribe("custom-request", func(message *nats.Msg) {
		assert.JSONEq(t, `{"FieldIn": "REQUEST"}`, string(message.Data))
		connection.Publish(message.Reply, []byte(`{"FieldOut": "RESPONSE"}`))
		requestSent <- true
	})
	assert.Nil(t, err)

	response, err := sender.Request(&customRequestProducer{
		&customRequest{"REQUEST"},
	})
	assert.Nil(t, err)
	assert.Exactly(t, customResponse{"RESPONSE"}, *response.(*customResponse))

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

type customRequestConsumer struct {
	Callback func(request *customRequest) *customResponse
}

func (consumer *customRequestConsumer) GetRequestType() communication.RequestType {
	return communication.RequestType("custom-response")
}

func (consumer *customRequestConsumer) NewRequest() (requestPtr interface{}) {
	return &customRequest{}
}

func (consumer *customRequestConsumer) Consume(requestPtr interface{}) (responsePtr interface{}, err error) {
	return consumer.Callback(requestPtr.(*customRequest)), nil
}

func TestCustomRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	requestReceived := make(chan bool)
	err := receiver.Respond(&customRequestConsumer{
		func(request *customRequest) *customResponse {
			assert.Equal(t, &customRequest{"REQUEST"}, request)
			requestReceived <- true
			return &customResponse{"RESPONSE"}
		},
	})
	assert.Nil(t, err)

	err = connection.PublishRequest("custom-response", "custom-reply", []byte(`{"FieldIn": "REQUEST"}`))
	assert.Nil(t, err)

	requestResponded := make(chan bool)
	_, err = connection.Subscribe("custom-reply", func(message *nats.Msg) {
		assert.JSONEq(t, `{"FieldOut": "RESPONSE"}`, string(message.Data))
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
