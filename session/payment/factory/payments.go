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

package factory

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/payment"
	"github.com/mysteriumnetwork/node/session/payment/noop"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/issuers"
	"github.com/pkg/errors"
)

// PaymentIssuerFactoryFunc returns a factory for payment issuer. It will be noop if the experimental payment flag is not set
func PaymentIssuerFactoryFunc(nodeOptions node.Options, signerFactory identity.SignerFactory) func(
	initialState promise.State,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentManager, error) {
	if !nodeOptions.ExperimentPayments {
		return noopPaymentIssuerFactory
	}
	return paymentIssuerFactory(signerFactory)
}

func noopPaymentIssuerFactory(initialState promise.State,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentManager, error) {
	return noop.NewSessionBalance(), nil

}

func paymentIssuerFactory(signerFactory identity.SignerFactory) func(
	initialState promise.State,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentManager, error) {
	return func(
		initialState promise.State,
		messageChan chan balance.Message,
		dialog communication.Dialog,
		consumer, provider identity.Identity) (connection.PaymentManager, error) {

		bl := balance.NewListener(messageChan)
		ps := promise.NewSender(dialog)
		issuer := issuers.NewLocalIssuer(signerFactory(consumer))
		tracker := promise.NewConsumerTracker(initialState, consumer, provider, issuer)
		payments := payment.NewSessionPayments(messageChan, ps, tracker)
		err := dialog.Receive(bl.GetConsumer())
		return payments, errors.Wrap(err, "fail to receive from consumer")
	}
}
