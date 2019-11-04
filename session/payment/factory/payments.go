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
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/payment"
	"github.com/mysteriumnetwork/node/session/pingpong/paydef"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/issuers"
	"github.com/pkg/errors"
)

// PromiseWaitTimeout is the time that the provider waits for the promise to arrive
const PromiseWaitTimeout = time.Second * 10

// BalanceSendPeriod is how often the provider will send balance messages to the consumer
const BalanceSendPeriod = time.Second * 20

// PaymentIssuerFactoryFunc returns a factory for payment issuer. It will be noop if the experimental payment flag is not set
func PaymentIssuerFactoryFunc(nodeOptions node.Options, signerFactory identity.SignerFactory) func(
	initialState promise.PaymentInfo,
	paymentDefinition paydef.PaymentRate,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentIssuer, error) {
	return paymentIssuerFactory(signerFactory)
}

func paymentIssuerFactory(signerFactory identity.SignerFactory) func(
	initialState promise.PaymentInfo,
	paymentDefinition paydef.PaymentRate,
	messageChan chan balance.Message,
	dialog communication.Dialog,
	consumer, provider identity.Identity) (connection.PaymentIssuer, error) {
	return func(
		initialState promise.PaymentInfo,
		paymentDefinition paydef.PaymentRate,
		messageChan chan balance.Message,
		dialog communication.Dialog,
		consumer, provider identity.Identity) (connection.PaymentIssuer, error) {

		bl := balance.NewListener(messageChan)
		ps := promise.NewSender(dialog)
		issuer := issuers.NewLocalIssuer(signerFactory(consumer))

		promiseState := mapInitialStateToPromiseState(initialState)
		tracker := promise.NewConsumerTracker(promiseState, consumer, provider, issuer)
		timeTracker := session.NewTracker(time.Now)
		amountCalc := session.AmountCalc{PaymentDef: paymentDefinition}

		balanceTracker := balance.NewBalanceTracker(&timeTracker, amountCalc, initialState.FreeCredit)
		payments := payment.NewSessionPayments(messageChan, ps, tracker, balanceTracker)
		err := dialog.Receive(bl.GetConsumer())
		return payments, errors.Wrap(err, "fail to receive from consumer")
	}
}

func mapInitialStateToPromiseState(initialState promise.PaymentInfo) promise.State {
	return promise.State{
		Seq:    initialState.LastPromise.SequenceID,
		Amount: initialState.LastPromise.Amount,
	}
}
