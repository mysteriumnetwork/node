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

	"github.com/mysteriumnetwork/node/communication"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
)

type customMessage struct {
	Field int
}

type customMessageProducer struct {
	Message *customMessage
}

func (producer *customMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("custom-message")
}

func (producer *customMessageProducer) Produce() (messagePtr interface{}) {
	return producer.Message
}

func TestMessageCustomSend(t *testing.T) {
	connection := StartConnectionMock()
	defer connection.Close()

	sender := &senderNATS{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	err := sender.Send(&customMessageProducer{&customMessage{123}})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"Field": 123}`, string(connection.GetLastMessage()))
}

func TestMessageCustomSendNull(t *testing.T) {
	connection := StartConnectionMock()
	defer connection.Close()

	sender := &senderNATS{
		connection: connection,
		codec:      communication.NewCodecJSON(),
	}

	err := sender.Send(&customMessageProducer{})
	assert.NoError(t, err)
	assert.JSONEq(t, `null`, string(connection.GetLastMessage()))
}

type customMessageConsumer struct {
	messageReceived chan interface{}
}

func (consumer *customMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return communication.MessageEndpoint("custom-message")
}

func (consumer *customMessageConsumer) NewMessage() (messagePtr interface{}) {
	return &customMessage{}
}

func (consumer *customMessageConsumer) Consume(messagePtr interface{}) error {
	consumer.messageReceived <- messagePtr
	return nil
}

func TestMessageCustomReceive(t *testing.T) {
	connection := StartConnectionMock()
	defer connection.Close()

	receiver := &receiverNATS{
		connection: connection,
		codec:      communication.NewCodecJSON(),
		subs:       make(map[string]*nats.Subscription),
	}

	consumer := &customMessageConsumer{messageReceived: make(chan interface{})}
	err := receiver.Receive(consumer)
	assert.NoError(t, err)

	connection.Publish("custom-message", []byte(`{"Field":123}`))
	message, err := connection.MessageWait(consumer.messageReceived)
	assert.NoError(t, err)
	assert.Exactly(t, customMessage{123}, *message.(*customMessage))
}
