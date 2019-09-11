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
	"github.com/mysteriumnetwork/payments/crypto"
)

// InvoiceRequest structure represents the invoice message that the provider sends to the consumer
type InvoiceRequest struct {
	Invoice crypto.Invoice `json:"invoice"`
}

const endpointInvoice = "session-invoice"
const messageEndpointInvoice = communication.MessageEndpoint(endpointInvoice)

// InvoiceSender is responsible for sending the invoice messages
type InvoiceSender struct {
	sender communication.Sender
}

// NewInvoiceSender returns a new instance of the invoice sender
func NewInvoiceSender(sender communication.Sender) *InvoiceSender {
	return &InvoiceSender{
		sender: sender,
	}
}

// Send sends the given invoice
func (is *InvoiceSender) Send(invoice crypto.Invoice) error {
	return is.sender.Send(&invoiceMessageProducer{Invoice: invoice})
}

// InvoiceListener listens for invoices
type InvoiceListener struct {
	invoiceMessageConsumer *invoiceMessageConsumer
}

// NewInvoiceListener returns a new instance of the invoice listener
func NewInvoiceListener(messageChan chan crypto.Invoice) *InvoiceListener {
	return &InvoiceListener{
		invoiceMessageConsumer: &invoiceMessageConsumer{
			queue: messageChan,
		},
	}
}

// GetConsumer returns the underlying invoice message consumer. Mostly here for the communication to work.
func (il *InvoiceListener) GetConsumer() *invoiceMessageConsumer {
	return il.invoiceMessageConsumer
}

// Consume handles requests from endpoint and replies with response
func (imc *invoiceMessageConsumer) Consume(requestPtr interface{}) (err error) {
	request := requestPtr.(*InvoiceRequest)
	imc.queue <- request.Invoice
	return nil
}

// Dialog boilerplate below, please ignore

type invoiceMessageConsumer struct {
	queue chan crypto.Invoice
}

// GetMessageEndpoint returns endpoint where to receive messages
func (imc *invoiceMessageConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointInvoice
}

// NewRequest creates struct where request from endpoint will be serialized
func (imc *invoiceMessageConsumer) NewMessage() (requestPtr interface{}) {
	return &InvoiceRequest{}
}

// invoiceMessageProducer
type invoiceMessageProducer struct {
	Invoice crypto.Invoice
}

// GetMessageEndpoint returns endpoint where to receive messages
func (imp *invoiceMessageProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return messageEndpointInvoice
}

// Produce produces a request message
func (imp *invoiceMessageProducer) Produce() (requestPtr interface{}) {
	return &InvoiceRequest{
		Invoice: imp.Invoice,
	}
}

// NewResponse returns a new response object
func (imp *invoiceMessageProducer) NewResponse() (responsePtr interface{}) {
	return nil
}
