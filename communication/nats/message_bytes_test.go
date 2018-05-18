/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
)

type bytesMessageProducer struct {
	Message []byte
}

func (producer *bytesMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("bytes-message")
}

func (producer *bytesMessageProducer) Produce() (messagePtr interface{}) {
	return producer.Message
}

type bytesMessageConsumer struct {
	messageReceived chan interface{}
}

func (consumer *bytesMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("bytes-message")
}

func (consumer *bytesMessageConsumer) NewMessage() (messagePtr interface{}) {
	var message []byte
	return &message
}

func (consumer *bytesMessageConsumer) Consume(messagePtr interface{}) error {
	consumer.messageReceived <- messagePtr
	return nil
}

func TestMessageBytesSend(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	sender := &senderNATS{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	err := sender.Send(
		&bytesMessageProducer{[]byte("123")},
	)
	assert.NoError(t, err)
	assert.Equal(t, []byte("123"), connection.GetLastMessage())
}

func TestMessageBytesReceive(t *testing.T) {
	connection := StartConnectionFake()
	defer connection.Close()

	receiver := &receiverNATS{
		connection: connection,
		codec:      communication.NewCodecBytes(),
	}

	consumer := &bytesMessageConsumer{messageReceived: make(chan interface{})}
	err := receiver.Receive(consumer)
	assert.NoError(t, err)

	connection.Publish("bytes-message", []byte("123"))
	message, err := connection.MessageWait(consumer.messageReceived)
	assert.NoError(t, err)
	assert.Equal(t, []byte("123"), *message.(*[]byte))
}
