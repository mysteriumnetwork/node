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
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/payments/crypto"
)

// InvoiceFactoryCreator returns a payment engine factory
func InvoiceFactoryCreator(dialog communication.Dialog, balanceSendPeriod, promiseTimeout time.Duration) func() (session.PaymentEngine, error) {
	return func() (session.PaymentEngine, error) {
		exchangeChan := make(chan crypto.ExchangeMessage, 1)
		listener := NewExchangeListener(exchangeChan)
		invoiceSender := NewInvoiceSender(dialog)
		err := dialog.Receive(listener.GetConsumer())
		if err != nil {
			return nil, err
		}
		paymentEngine := NewInvoiceTracker(dialog.PeerID(), invoiceSender, balanceSendPeriod, exchangeChan, promiseTimeout)
		return paymentEngine, nil
	}
}

// ExchangeMessageFactoryCreator returns a payment engine factory
func ExchangeMessageFactoryCreator(keystore *keystore.KeyStore) connection.PaymentEngineFactory {
	return func(invoice chan crypto.Invoice, dialog communication.Dialog, consumer identity.Identity) (connection.PaymentIssuer, error) {
		sender := NewExchangeSender(dialog)
		listener := NewInvoiceListener(invoice)
		err := dialog.Receive(listener.GetConsumer())
		if err != nil {
			return nil, err
		}
		return NewExchangeMessageTracker(invoice, sender, keystore, consumer), nil
	}
}
