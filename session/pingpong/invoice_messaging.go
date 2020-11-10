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
	"context"
	"time"

	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

// InvoiceRequest structure represents the invoice message that the provider sends to the consumer.
type InvoiceRequest struct {
	Invoice crypto.Invoice `json:"invoice"`
}

// InvoiceSender is responsible for sending the invoice messages.
type InvoiceSender struct {
	ch p2p.ChannelSender
}

// NewInvoiceSender returns a new instance of the invoice sender.
func NewInvoiceSender(ch p2p.ChannelSender) *InvoiceSender {
	return &InvoiceSender{
		ch: ch,
	}
}

// Send sends the given invoice.
func (is *InvoiceSender) Send(invoice crypto.Invoice) error {
	pInvoice := &pb.Invoice{
		AgreementID:    invoice.AgreementID.Text(bigIntBase),
		AgreementTotal: invoice.AgreementTotal.Text(bigIntBase),
		TransactorFee:  invoice.TransactorFee.Text(bigIntBase),
		Hashlock:       invoice.Hashlock,
		Provider:       invoice.Provider,
		ChainID:        invoice.ChainID,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicPaymentInvoice, pInvoice.String())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := is.ch.Send(ctx, p2p.TopicPaymentInvoice, p2p.ProtoMessage(pInvoice))
	return err
}
