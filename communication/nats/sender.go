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
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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
		err = errors.Wrapf(err, "failed to encode message '%s'", messageTopic)
		return err
	}

	if err := sender.connection.Check(); err != nil {
		log.Warn().Err(err).Msg("Connection failed")
	}

	log.WithLevel(levelFor(messageTopic)).Msgf("Message %q sending: %s", messageTopic, messageData)
	err = sender.connection.Publish(messageTopic, messageData)
	if err != nil {
		err = errors.Wrapf(err, "failed to send message '%s'", messageTopic)
		return err
	}

	return nil
}

func (sender *senderNATS) Request(producer communication.RequestProducer) (responsePtr interface{}, err error) {
	requestTopic := sender.messageTopic + string(producer.GetRequestEndpoint())
	responsePtr = producer.NewResponse()

	requestData, err := sender.codec.Pack(producer.Produce())
	if err != nil {
		err = errors.Wrapf(err, "failed to pack request '%s'", requestTopic)
		return
	}

	if err := sender.connection.Check(); err != nil {
		log.Warn().Err(err).Msg("Connection failed")
	}

	log.WithLevel(levelFor(requestTopic)).Msgf("Request %q sending: %s", requestTopic, requestData)
	msg, err := sender.connection.Request(requestTopic, requestData, sender.timeoutRequest)
	if err != nil {
		err = errors.Wrapf(err, "failed to send request '%s'", requestTopic)
		return
	}

	log.Debug().Msgf("Received response for %q: %s", requestTopic, msg.Data)
	err = sender.codec.Unpack(msg.Data, responsePtr)
	if err != nil {
		err = errors.Wrapf(err, "failed to unpack response '%s'", requestTopic)
		log.Error().Err(err).Msg("")
		return
	}

	return responsePtr, nil
}
