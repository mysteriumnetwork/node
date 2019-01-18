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

// Package balance is responsible for handling balance related communications and keeping track of the current balance.
package balance

import (
	"github.com/mysteriumnetwork/node/communication"
)

// Request structure represents message that the provider sends for the consumer to keep track of the balance
type Request struct {
	BalanceMessage Message `json:"balanceMessage"`
}

// Message shows the balance and the current sequence id(of the promise)
type Message struct {
	Balance    uint64 `json:"balance"`
	SequenceID uint64 `json:"sequenceID"`
}

const endpointBalance = "session-balance"
const messageEndpointBalance = communication.MessageEndpoint(endpointBalance)

// Sender is responsible for sending the balance messages
type Sender struct {
	sender communication.Sender
}

// NewBalanceSender returns a new instance of the balance sennder
func NewBalanceSender(sender communication.Sender) *Sender {
	return &Sender{
		sender: sender,
	}
}

// Send sends the given balance message
func (bs *Sender) Send(bm Message) error {
	err := bs.sender.Send(&balanceMessageProducer{BalanceMessage: bm})
	return err
}

// Listener listens for balance messages
type Listener struct {
	balanceMessageConsumer *balanceMessageConsumer
}

// NewListener returns a new instance of the balance listener
func NewListener() *Listener {
	return &Listener{
		balanceMessageConsumer: &balanceMessageConsumer{
			queue: make(chan Message, 1),
		},
	}
}

// Listen returns a read only channel where all the balance messages will be sent to
func (bl *Listener) Listen() <-chan Message {
	return bl.balanceMessageConsumer.queue
}

// GetConsumer returns the underlying balance message consumer. Mostly here for the communication to work.
func (bl *Listener) GetConsumer() *balanceMessageConsumer {
	return bl.balanceMessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (bmc *balanceMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*Request)
	bmc.queue <- request.BalanceMessage
	return nil
}

// Dialog boilerplate below, please ignore

type balanceMessageConsumer struct {
	queue chan Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (bmc *balanceMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointBalance
}

// NewRequest creates struct where request from endpoint will be serialized
func (bmc *balanceMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &Request{}
}

// balanceMessageProducer
type balanceMessageProducer struct {
	BalanceMessage Message
}

// GetMessageEndpoint returns endpoint where to receive messages
func (bmp *balanceMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointBalance
}

// Produce produces a request message
func (bmp *balanceMessageProducer) Produce() (requestPtr interface{}) {
	return &Request{
		BalanceMessage: Message{},
	}
}

// NewResponse returns a new response object
func (bmp *balanceMessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
