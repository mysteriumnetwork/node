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

func (request customRequest) ProduceMessage() []byte {
	messageBody, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	return messageBody
}

type customResponse struct {
	FieldOut string
}

func (response *customResponse) ConsumeMessage(messageBody []byte) {
	err := json.Unmarshal(messageBody, response)
	if err != nil {
		panic(err)
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
		customRequest{"REQUEST"},
		&response,
	)
	assert.Nil(t, err)
	assert.Equal(t, customResponse{"RESPONSE"}, response)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}
