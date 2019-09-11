/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package promise

import (
	"github.com/mysteriumnetwork/node/communication"
)

// Request structure represents message from service consumer to send a promise
type Request struct {
	Message Message `json:"promiseMessage"`
}

// Message is the payload we send to the provider
type Message struct {
	Amount     uint64 `json:"amount"`
	SequenceID uint64 `json:"sequenceID"`
	Signature  string `json:"signature"`
}

// PaymentInfo represents the payment information that the provider has about the consumer
type PaymentInfo struct {
	LastPromise LastPromise `json:"lastPromise"`
	FreeCredit  uint64      `json:"freeCredit"`
	Supports    string      `json:"supported"`
}

// LastPromise represents the last known promise to the provider
// If the seqid and amount are 0 - there's no known info
type LastPromise struct {
	SequenceID uint64 `json:"sequenceID"`
	Amount     uint64 `json:"amount"`
}

const endpointPromise = "session-promise"
const messageEndpointPromise = communication.MessageEndpoint(endpointPromise)

// Sender is responsible for sending the promise messages
type Sender struct {
	sender communication.Sender
}

// NewSender returns a new instance of promise sender
func NewSender(sender communication.Sender) *Sender {
	return &Sender{
		sender: sender,
	}
}

// Send send the given promise message
func (ps *Sender) Send(pm Message) error {
	return ps.sender.Send(&MessageProducer{Message: pm})
}

// Listener listens for promise messages
type Listener struct {
	MessageConsumer *MessageConsumer
}

// NewListener returns a new instance of promise listener
func NewListener(promiseChan chan Message) *Listener {
	return &Listener{
		MessageConsumer: &MessageConsumer{
			queue: promiseChan,
		},
	}
}

// GetConsumer gets the underlying consumer from the listener
func (pl *Listener) GetConsumer() *MessageConsumer {
	return pl.MessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (pmc *MessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*Request)
	pmc.queue <- request.Message
	return nil
}

// Dialog boilerplate below, please ignore

// MessageConsumer is responsible for consuming the messages
type MessageConsumer struct {
	queue chan Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *MessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

// NewMessage creates struct where request from endpoint will be serialized
func (pmc *MessageConsumer) NewMessage() (requestPtr interface{}) {
	return &Request{}
}

// MessageProducer handles the production of messages from the provider side
type MessageProducer struct {
	Message Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmp *MessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointPromise
}

// Produce creates the actual message
func (pmp *MessageProducer) Produce() (requestPtr interface{}) {
	return &Request{
		Message: pmp.Message,
	}
}

// NewResponse creates a new empty response
func (pmp *MessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
