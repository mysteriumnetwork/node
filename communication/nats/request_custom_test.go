/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package nats

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
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

func (producer *customRequestProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return communication.RequestEndpoint("custom-request")
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

	sender := &senderNATS{
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

func (consumer *customRequestConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return communication.RequestEndpoint("custom-response")
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

	receiver := &receiverNATS{
		connection: connection,
		codec:      communication.NewCodecJSON(),
		subs:       make(map[string]*nats.Subscription),
	}

	consumer := &customRequestConsumer{}
	err := receiver.Respond(consumer)
	assert.NoError(t, err)

	response, err := connection.Request("custom-response", []byte(`{"FieldIn": "REQUEST"}`), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, &customRequest{"REQUEST"}, consumer.requestReceived)
	assert.JSONEq(t, `{"FieldOut": "RESPONSE"}`, string(response.Data))
}
