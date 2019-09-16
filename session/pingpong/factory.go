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
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	payment_factory "github.com/mysteriumnetwork/node/session/payment/factory"
	"github.com/mysteriumnetwork/node/session/promise"
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

// BackwardsCompatibleExchangeFactoryFunc returns a backwards compatible version of the exchange factory
func BackwardsCompatibleExchangeFactoryFunc(keystore *keystore.KeyStore, options node.Options, signer identity.SignerFactory) func(paymentInfo *promise.PaymentInfo,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentIssuer, error) {
	return func(paymentInfo *promise.PaymentInfo,
		dialog communication.Dialog,
		consumer, provider identity.Identity) (connection.PaymentIssuer, error) {
		var promiseState promise.PaymentInfo
		payment := dto.PaymentPerTime{
			Price: money.Money{
				Currency: money.CurrencyMyst,
				Amount:   uint64(0),
			},
			Duration: time.Minute,
		}
		var useNewPayments bool
		if paymentInfo != nil {
			promiseState.FreeCredit = paymentInfo.FreeCredit
			promiseState.LastPromise = paymentInfo.LastPromise

			// if the server indicates that it will launch the new payments, so should we
			if paymentInfo.Supports == string(session.PaymentVersionV2) {
				useNewPayments = true
			}
		}
		var payments connection.PaymentIssuer
		if useNewPayments {
			log.Info("using new payments")
			invoices := make(chan crypto.Invoice)
			sender := NewExchangeSender(dialog)
			listener := NewInvoiceListener(invoices)
			err := dialog.Receive(listener.GetConsumer())
			if err != nil {
				return nil, err
			}
			payments = NewExchangeMessageTracker(invoices, sender, keystore, consumer)
		} else {
			log.Info("using old payments")
			messageChan := make(chan balance.Message, 1)
			pFunc := payment_factory.PaymentIssuerFactoryFunc(options, signer)
			p, err := pFunc(promiseState, payment, messageChan, dialog, consumer, provider)
			if err != nil {
				return nil, err
			}
			payments = p
		}
		return payments, nil
	}
}
