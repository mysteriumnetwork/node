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
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/nats-io/go-nats"
)

const receiverLogPrefix = "[NATS.Receiver] "

// NewReceiver constructs new Receiver's instance which works thru NATS connection.
// Codec packs/unpacks messages to byte payloads.
// Topic (optional) if need to send messages prefixed topic.
func NewReceiver(connection Connection, codec communication.Codec, topic string) *receiverNATS {
	return &receiverNATS{
		connection:   connection,
		codec:        codec,
		messageTopic: topic + ".",
	}
}

type receiverNATS struct {
	connection   Connection
	codec        communication.Codec
	messageTopic string
}

func (receiver *receiverNATS) Receive(consumer communication.MessageConsumer) error {

	messageTopic := receiver.messageTopic + string(consumer.GetMessageEndpoint())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(receiverLogPrefix, fmt.Sprintf("Message '%s' received: %s", messageTopic, msg.Data))
		messagePtr := consumer.NewMessage()
		err := receiver.codec.Unpack(msg.Data, messagePtr)
		if err != nil {
			err = fmt.Errorf("failed to unpack message '%s'. %s", messageTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		err = consumer.Consume(messagePtr)
		if err != nil {
			err = fmt.Errorf("failed to process message '%s'. %s", messageTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}
	}

	_, err := receiver.connection.Subscribe(messageTopic, messageHandler)
	if err != nil {
		err = fmt.Errorf("failed subscribe message '%s'. %s", messageTopic, err)
		return err
	}

	return nil
}

func (receiver *receiverNATS) Respond(consumer communication.RequestConsumer) error {
	requestTopic := receiver.messageTopic + string(consumer.GetRequestEndpoint())

	messageHandler := func(msg *nats.Msg) {
		log.Debug(receiverLogPrefix, fmt.Sprintf("Request '%s' received: %s", requestTopic, msg.Data))
		requestPtr := consumer.NewRequest()
		err := receiver.codec.Unpack(msg.Data, requestPtr)
		if err != nil {
			err = fmt.Errorf("failed to unpack request '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		response, err := consumer.Consume(requestPtr)
		if err != nil {
			err = fmt.Errorf("failed to process request '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		responseData, err := receiver.codec.Pack(response)
		if err != nil {
			err = fmt.Errorf("failed to pack response '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}

		log.Debug(receiverLogPrefix, fmt.Sprintf("Request '%s' response: %s", requestTopic, responseData))
		err = receiver.connection.Publish(msg.Reply, responseData)
		if err != nil {
			err = fmt.Errorf("failed to send response '%s'. %s", requestTopic, err)
			log.Error(receiverLogPrefix, err)
			return
		}
	}

	log.Debug(receiverLogPrefix, fmt.Sprintf("Request '%s' topic has been subscribed to", requestTopic))

	_, err := receiver.connection.Subscribe(requestTopic, messageHandler)
	// TODO: nats.Subscription.Unsubscribe() should be called when topic is no longer needed
	//  session-create and session-destroy topic should be cleared after session-destroy message is received
	//    or session timeouts (promise processor detects session timeout)
	// scope of "session-create topic deduplication #533"
	if err != nil {
		err = fmt.Errorf("failed subscribe request '%s'. %s", requestTopic, err)
		return err
	}

	return nil
}
