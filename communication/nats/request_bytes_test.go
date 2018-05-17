/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type bytesRequestProducer struct {
	Request []byte
}

func (producer *bytesRequestProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return communication.RequestEndpoint("bytes-request")
}

func (producer *bytesRequestProducer) NewResponse() (responsePtr interface{}) {
	var response []byte
	return &response
}

func (producer *bytesRequestProducer) Produce() (requestPtr interface{}) {
	return producer.Request
}

func TestBytesRequest(t *testing.T) {
	connection := StartConnectionFake()
	connection.MockResponse("bytes-request", []byte("RESPONSE"))
	defer connection.Close()

	sender := &senderNATS{
		connection:     connection,
		codec:          communication.NewCodecBytes(),
		timeoutRequest: 100 * time.Millisecond,
	}

	response, err := sender.Request(&bytesRequestProducer{
		[]byte("REQUEST"),
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("REQUEST"), connection.GetLastRequest())
	assert.Equal(t, []byte("RESPONSE"), *response.(*[]byte))
}

type bytesRequestConsumer struct {
	requestReceived interface{}
}

func (consumer *bytesRequestConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return communication.RequestEndpoint("bytes-response")
}

func (consumer *bytesRequestConsumer) NewRequest() (requestPtr interface{}) {
	var request []byte
	return &request
}

func (consumer *bytesRequestConsumer) Consume(requestPtr interface{}) (responsePtr interface{}, err error) {
	consumer.requestReceived = requestPtr
	return []byte("RESPONSE"), nil
}

func TestBytesRespond(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	receiver := &receiverNATS{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	consumer := &bytesRequestConsumer{}
	err := receiver.Respond(consumer)
	assert.NoError(t, err)

	response, err := connection.Request("bytes-response", []byte("REQUEST"), time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, []byte("REQUEST"), *consumer.requestReceived.(*[]byte))
	assert.Equal(t, []byte("RESPONSE"), response.Data)
}
