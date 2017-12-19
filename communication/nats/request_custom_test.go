package nats

import (
	"github.com/mysterium/node/communication"
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
	connection := StartConnectionFake()
	connection.MockResponse("custom-request", []byte(`{"FieldOut": "RESPONSE"}`))
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		codec:          communication.NewCodecJSON(),
		timeoutRequest: 100 * time.Millisecond,
	}

	response, err := sender.Request(&customRequestProducer{
		&customRequest{"REQUEST"},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"FieldIn": "REQUEST"}`, string(connection.GetLastRequest()))
	assert.Exactly(t, customResponse{"RESPONSE"}, *response.(*customResponse))
}

type customRequestConsumer struct {
	requestReceived interface{}
}

func (consumer *customRequestConsumer) GetRequestType() communication.RequestType {
	return communication.RequestType("custom-response")
}

func (consumer *customRequestConsumer) NewRequest() (requestPtr interface{}) {
	return &customRequest{}
}

func (consumer *customRequestConsumer) Consume(requestPtr interface{}) (responsePtr interface{}, err error) {
	consumer.requestReceived = requestPtr
	return &customResponse{"RESPONSE"}, nil
}

func TestCustomRespond(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	consumer := &customRequestConsumer{}
	err := receiver.Respond(consumer)
	assert.NoError(t, err)

	response, err := connection.Request("custom-response", []byte(`{"FieldIn": "REQUEST"}`), time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, &customRequest{"REQUEST"}, consumer.requestReceived)
	assert.JSONEq(t, `{"FieldOut": "RESPONSE"}`, string(response.Data))
}
