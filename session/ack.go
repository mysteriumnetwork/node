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

package session

import (
	"github.com/mysteriumnetwork/node/communication"
)

const endpointSessionAck = communication.RequestEndpoint("session-acknowledge")

// AckProducer allows for creating of ack messages
type AckProducer struct {
	Payload interface{}
}

// AckResponse is the response we send for an ack message
type AckResponse struct {
}

// AckRequest represents request we send for an ack
type AckRequest struct {
	Request interface{} `json:"request"`
}

// GetRequestEndpoint returns the ack endpoint
func (sap *AckProducer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionAck
}

// NewResponse creates a new ack response
func (sap *AckProducer) NewResponse() (responsePtr interface{}) {
	return &AckResponse{}
}

// Produce creates a new ack request
func (sap *AckProducer) Produce() (requestPtr interface{}) {
	return &AckRequest{
		Request: sap.Payload,
	}
}

// AckConsumer knows how to behave when an ack message is received
type AckConsumer struct {
	ack ConfigReceiver
}

// GetRequestEndpoint returns the ack endpoint
func (sac *AckConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionAck
}

// NewRequest creates a new session ack request
func (sac *AckConsumer) NewRequest() (requestPtr interface{}) {
	var request AckRequest
	return &request
}

// Consume consumes the ack message
func (sac *AckConsumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	err = sac.ack(requestPtr)
	return
}
