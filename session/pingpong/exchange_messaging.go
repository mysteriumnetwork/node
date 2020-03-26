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

package pingpong

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/payments/crypto"
)

// ExchangeRequest structure represents message from service consumer to send a an exchange message.
type ExchangeRequest struct {
	Message crypto.ExchangeMessage `json:"exchangeMessage"`
}

const endpointExchange = "session-exchange"
const messageEndpointExchange = communication.MessageEndpoint(endpointExchange)

// ExchangeSender is responsible for sending the exchange messages.
type ExchangeSender struct {
	sender communication.Sender
	ch     *p2p.Channel
}

// NewExchangeSender returns a new instance of exchange message sender.
func NewExchangeSender(sender communication.Sender, ch *p2p.Channel) *ExchangeSender {
	return &ExchangeSender{
		ch:     ch,
		sender: sender,
	}
}

// Send sends the given exchange message.
func (es *ExchangeSender) Send(em crypto.ExchangeMessage) error {
	if es.ch == nil { // TODO this block should go away once p2p communication will replace communication dialog.
		return es.sender.Send(&ExchangeMessageProducer{Message: em})
	}
	pMessage := p2p.ProtoMessage(&pb.ExchangeMessage{
		Promise: &pb.Promise{
			ChannelID: em.Promise.ChannelID,
			Amount:    em.Promise.Amount,
			Fee:       em.Promise.Fee,
			Hashlock:  em.Promise.Hashlock,
			R:         em.Promise.R,
			Signature: em.Promise.Signature,
		},
		AgreementID:    em.AgreementID,
		AgreementTotal: em.AgreementTotal,
		Provider:       em.Provider,
		Signature:      em.Signature,
	})
	_, err := es.ch.Send(p2p.TopicPaymentMessage, pMessage)
	return err
}

// ExchangeListener listens for exchange messages.
type ExchangeListener struct {
	MessageConsumer *ExchangeMessageConsumer
}

// NewExchangeListener returns a new instance of exchange message listener.
func NewExchangeListener(exchangeChan chan crypto.ExchangeMessage) *ExchangeListener {
	return &ExchangeListener{
		MessageConsumer: &ExchangeMessageConsumer{
			queue: exchangeChan,
		},
	}
}

// GetConsumer gets the underlying consumer from the listener.
func (el *ExchangeListener) GetConsumer() *ExchangeMessageConsumer {
	return el.MessageConsumer
}

// Consume handles requests from endpoint and replies with response.
func (emc *ExchangeMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*ExchangeRequest)
	emc.queue <- request.Message
	return nil
}

// Dialog boilerplate below, please ignore

// ExchangeMessageConsumer is responsible for consuming the exchange messages.
type ExchangeMessageConsumer struct {
	queue chan crypto.ExchangeMessage
}

// GetMessageEndpoint returns endpoint where to receive messages.
func (emc *ExchangeMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointExchange
}

// NewMessage creates struct where request from endpoint will be serialized.
func (emc *ExchangeMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &ExchangeRequest{}
}

// ExchangeMessageProducer handles the production of messages from the consumer side.
type ExchangeMessageProducer struct {
	Message crypto.ExchangeMessage
}

// GetMessageEndpoint returns endpoint where to receive messages.
func (emp *ExchangeMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointExchange
}

// Produce creates the actual message.
func (emp *ExchangeMessageProducer) Produce() (requestPtr interface{}) {
	return &ExchangeRequest{
		Message: emp.Message,
	}
}

// NewResponse creates a new empty response.
func (emp *ExchangeMessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
