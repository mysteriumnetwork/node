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
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/balance"
	payment_factory "github.com/mysteriumnetwork/node/session/payment/factory"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

// DefaultAccountantFailureCount defines how many times we're allowed to fail to reach accountant in a row before announcing the failure.
const DefaultAccountantFailureCount uint64 = 10

// DefaultPaymentInfo represents the default payment info for the alpha release
var DefaultPaymentInfo = dto.PaymentRate{
	Price:    money.NewMoney(100000, money.CurrencyMyst),
	Duration: 1 * time.Minute,
}

// InvoiceFactoryCreator returns a payment engine factory.
func InvoiceFactoryCreator(
	dialog communication.Dialog,
	balanceSendPeriod, promiseTimeout time.Duration,
	invoiceStorage providerInvoiceStorage,
	paymentInfo dto.PaymentRate,
	accountantCaller accountantCaller,
	accountantPromiseStorage accountantPromiseStorage,
	registryAddress string,
	channelImplementationAddress string,
	maxAccountantFailureCount uint64,
	maxAllowedAccountantFee uint16,
	blockchainHelper bcHelper,
	publisher eventbus.Publisher,
	feeProvider feeProvider,
) func(identity.Identity, identity.Identity) (session.PaymentEngine, error) {
	return func(providerID identity.Identity, accountantID identity.Identity) (session.PaymentEngine, error) {
		exchangeChan := make(chan crypto.ExchangeMessage, 1)
		listener := NewExchangeListener(exchangeChan)
		invoiceSender := NewInvoiceSender(dialog)
		err := dialog.Receive(listener.GetConsumer())
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(time.Now)
		deps := InvoiceTrackerDeps{
			Peer:                       dialog.PeerID(),
			PeerInvoiceSender:          invoiceSender,
			InvoiceStorage:             invoiceStorage,
			TimeTracker:                &timeTracker,
			ChargePeriod:               balanceSendPeriod,
			ExchangeMessageChan:        exchangeChan,
			ExchangeMessageWaitTimeout: promiseTimeout,
			PaymentInfo:                paymentInfo,
			ProviderID:                 providerID,
			AccountantCaller:           accountantCaller,
			AccountantPromiseStorage:   accountantPromiseStorage,
			AccountantID:               accountantID,
			Registry:                   registryAddress,
			MaxAccountantFailureCount:  maxAccountantFailureCount,
			MaxAllowedAccountantFee:    maxAllowedAccountantFee,
			BlockchainHelper:           blockchainHelper,
			Publisher:                  publisher,
			FeeProvider:                feeProvider,
			ChannelAddressCalculator:   NewChannelAddressCalculator(accountantID.Address, channelImplementationAddress, registryAddress),
		}
		paymentEngine := NewInvoiceTracker(deps)
		return paymentEngine, nil
	}
}

// BackwardsCompatibleExchangeFactoryFunc returns a backwards compatible version of the exchange factory.
func BackwardsCompatibleExchangeFactoryFunc(
	keystore *keystore.KeyStore,
	options node.Options,
	signer identity.SignerFactory,
	invoiceStorage consumerInvoiceStorage,
	totalStorage consumerTotalsStorage,
	channelImplementation string,
	registryAddress string, publisher eventbus.Publisher) func(paymentInfo *promise.PaymentInfo,
	dialog communication.Dialog,
	consumer, provider, accountant identity.Identity) (connection.PaymentIssuer, error) {
	return func(paymentInfo *promise.PaymentInfo,
		dialog communication.Dialog,
		consumer, provider, accountant identity.Identity) (connection.PaymentIssuer, error) {
		var promiseState promise.PaymentInfo
		payment := dto.PaymentRate{
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
			log.Info().Msg("Using new payments")
			invoices := make(chan crypto.Invoice)
			listener := NewInvoiceListener(invoices)
			err := dialog.Receive(listener.GetConsumer())
			if err != nil {
				return nil, err
			}
			timeTracker := session.NewTracker(time.Now)
			deps := ExchangeMessageTrackerDeps{
				InvoiceChan:               invoices,
				PeerExchangeMessageSender: NewExchangeSender(dialog),
				ConsumerInvoiceStorage:    invoiceStorage,
				ConsumerTotalsStorage:     totalStorage,
				TimeTracker:               &timeTracker,
				Ks:                        keystore,
				Identity:                  consumer,
				Peer:                      dialog.PeerID(),
				PaymentInfo:               DefaultPaymentInfo,
				ChannelAddressCalculator:  NewChannelAddressCalculator(accountant.Address, channelImplementation, registryAddress),
				Publisher:                 publisher,
			}
			payments = NewExchangeMessageTracker(deps)
		} else {
			log.Info().Msg("Using old payments")
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
