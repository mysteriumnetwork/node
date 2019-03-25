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
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
)

const senderLogPrefix = "[NATS.Sender] "

// NewSender constructs new Sender's instance which works thru NATS connection.
// Codec packs/unpacks messages to byte payloads.
// Topic (optional) if need to send messages prefixed topic.
func NewSender(connection Connection, codec communication.Codec, topic string) *senderNATS {
	return &senderNATS{
		connection:     connection,
		codec:          codec,
		timeoutRequest: 10 * time.Second,
		messageTopic:   topic + ".",
	}
}

type senderNATS struct {
	connection     Connection
	codec          communication.Codec
	timeoutRequest time.Duration
	messageTopic   string
}

func (sender *senderNATS) Send(producer communication.MessageProducer) error {
	messageTopic := sender.messageTopic + string(producer.GetMessageEndpoint())

	messageData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("failed to encode message '%s'. %s", messageTopic, err)
		return err
	}

	if err := sender.connection.Check(); err != nil {
		log.Warn(senderLogPrefix, "Connection failed: ", err)
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Message '%s' sending: %s", messageTopic, messageData))
	err = sender.connection.Publish(messageTopic, messageData)
	if err != nil {
		err = fmt.Errorf("failed to send message '%s'. %s", messageTopic, err)
		return err
	}

	return nil
}

func (sender *senderNATS) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	requestTopic := sender.messageTopic + string(producer.GetRequestEndpoint())
	responsePtr = producer.NewResponse()

	requestData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = fmt.Errorf("failed to pack request '%s'. %s", requestTopic, err)
		return
	}

	if err := sender.connection.Check(); err != nil {
		log.Warn(senderLogPrefix, "Connection failed: ", err)
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Request '%s' sending: %s", requestTopic, requestData))
	msg, err := sender.connection.Request(requestTopic, requestData, sender.timeoutRequest)
	if err != nil {
		err = fmt.Errorf("failed to send request '%s'. %s", requestTopic, err)
		return
	}

	log.Debug(senderLogPrefix, fmt.Sprintf("Received response for '%s': %s", requestTopic, msg.Data))
	err = sender.codec.Unpack(msg.Data, responsePtr)
	if err != nil {
		err = fmt.Errorf("failed to unpack response '%s'. %s", requestTopic, err)
		log.Error(receiverLogPrefix, err)
		return
	}

	return responsePtr, nil
}
