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

func customRequestPack(request customRequest) communication.MessagePacker {
	return func() []byte {
		data, err := json.Marshal(request)
		if err != nil {
			panic(err)
		}
		return data
	}
}

func customResponseUnpack(response *customResponse) communication.MessageUnpacker {
	return func(data []byte) {
		err := json.Unmarshal(data, &response)
		if err != nil {
			panic(err)
		}
	}
}

func TestCustomRequest(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	sender := &senderNats{
		connection:     connection,
		timeoutRequest: 100 * time.Millisecond,
	}

	requestSent := make(chan bool)
	_, err := connection.Subscribe("custom-request", func(message *nats.Msg) {
		assert.JSONEq(t, `{"FieldIn": "REQUEST"}`, string(message.Data))
		connection.Publish(message.Reply, []byte(`{"FieldOut": "RESPONSE"}`))
		requestSent <- true
	})
	assert.Nil(t, err)

	response := customResponse{}
	err = sender.Request(
		communication.RequestType("custom-request"),
		customRequestPack(customRequest{"REQUEST"}),
		customResponseUnpack(&response),
	)
	assert.Nil(t, err)
	assert.Equal(t, customResponse{"RESPONSE"}, response)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

type customResponder struct {
	Callback func(request customRequest) customResponse
}

func (responder customResponder) ConsumeRequest(requestBody []byte) []byte {
	var request customRequest
	err := json.Unmarshal(requestBody, &request)
	if err != nil {
		panic(err)
	}

	response := responder.Callback(request)

	responseBody, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	return responseBody
}

func TestCustomRespond(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	receiver := &receiverNats{connection: connection}

	requestReceived := make(chan bool)
	err := receiver.Respond(
		communication.RequestType("custom-response"),
		customResponder{func(request customRequest) customResponse {
			assert.Equal(t, customRequest{"REQUEST"}, request)
			requestReceived <- true
			return customResponse{"RESPONSE"}
		}},
	)
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
