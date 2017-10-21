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

func (message customRequest) Pack() ([]byte, error) {
	return json.Marshal(message)
}

func (message *customRequest) Unpack(data []byte) error {
	return json.Unmarshal(data, message)
}

type customResponse struct {
	FieldOut string
}

func (message customResponse) Pack() ([]byte, error) {
	return json.Marshal(message)
}

func (message *customResponse) Unpack(data []byte) error {
	return json.Unmarshal(data, message)
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

	var response customResponse
	err = sender.Request(
		communication.RequestType("custom-request"),
		customRequest{"REQUEST"},
		&response,
	)
	assert.Nil(t, err)
	assert.Exactly(t, customResponse{"RESPONSE"}, response)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}

func customRequestHandler(callback func(*customRequest) *customResponse) communication.RequestHandler {
	return func(requestData []byte) []byte {
		var request customRequest
		err := request.Unpack(requestData)
		if err != nil {
			panic(err)
		}

		response := callback(&request)

		responseData, err := response.Pack()
		if err != nil {
			panic(err)
		}
		return responseData
	}
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
		customRequestHandler(func(request *customRequest) *customResponse {
			assert.Equal(t, &customRequest{"REQUEST"}, request)
			requestReceived <- true
			return &customResponse{"RESPONSE"}
		}),
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
