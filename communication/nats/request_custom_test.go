package nats

import (
	"encoding/json"
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

func requestCustom(sender communication.Sender, request *customRequest) (response *customResponse, err error) {
	err = sender.Request(&communication.RequestPacker{
		RequestType: "custom-request",
		RequestPack: func() ([]byte, error) {
			return json.Marshal(request)
		},
		ResponseUnpack: func(responseData []byte) error {
			return json.Unmarshal(responseData, &response)
		},
	})
	return response, err
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

	response, err := requestCustom(sender, &customRequest{"REQUEST"})
	assert.Nil(t, err)
	assert.Exactly(t, &customResponse{"RESPONSE"}, response)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

func respondCustom(receiver communication.Receiver, callback func(request *customRequest) *customResponse) error {
	var request *customRequest
	var response *customResponse

	return receiver.Respond(&communication.RequestUnpacker{
		RequestType: "custom-response",
		RequestUnpack: func(requestData []byte) error {
			return json.Unmarshal(requestData, &request)
		},
		ResponsePack: func() ([]byte, error) {
			return json.Marshal(response)
		},
		Invoke: func() error {
			response = callback(request)
			return nil
		},
	})
}

func TestCustomRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{
		connection: connection,
	}

	requestReceived := make(chan bool)
	err := respondCustom(receiver, func(request *customRequest) *customResponse {
		assert.Equal(t, &customRequest{"REQUEST"}, request)
		requestReceived <- true
		return &customResponse{"RESPONSE"}
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
