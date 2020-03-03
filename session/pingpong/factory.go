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
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// PromiseWaitTimeout is the time that the provider waits for the promise to arrive
const PromiseWaitTimeout = time.Second * 50

// InvoiceSendPeriod is how often the provider will send invoice messages to the consumer
const InvoiceSendPeriod = time.Second * 60

// DefaultAccountantFailureCount defines how many times we're allowed to fail to reach accountant in a row before announcing the failure.
const DefaultAccountantFailureCount uint64 = 10

// DefaultPaymentMethod represents the the default payment method of time + bytes.
// The rate is frozen at 0.07MYSTT per GiB of data transferred and 0.0005MYSTT/minute.
// Since the price is calculated based on the rate and price, for 1 GiB we need:
// 0.07 * 100 000 000 / 50 000 = 140 chunks.
// 1024 * 1024 * 1024(or 1 GiB)  / 140 ~= 7669584.
// Therefore, for reach 7669584 bytes transferred, we'll pay 0.0005 MYSTT.
var DefaultPaymentMethod = PaymentMethod{
	Price:    money.NewMoney(50000, money.CurrencyMyst),
	Duration: time.Minute,
	Type:     "BYTES_TRANSFERRED_WITH_TIME",
	Bytes:    7669584,
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	Price    money.Money   `json:"price"`
	Duration time.Duration `json:"duration"`
	Bytes    uint64        `json:"bytes"`
	Type     string        `json:"type"`
}

// GetPrice returns the payment methods price
func (pm PaymentMethod) GetPrice() money.Money {
	return pm.Price
}

// GetType gets the payment methods type
func (pm PaymentMethod) GetType() string {
	return pm.Type
}

// GetRate returns the payment rate for the method
func (pm PaymentMethod) GetRate() market.PaymentRate {
	return market.PaymentRate{PerByte: pm.Bytes, PerTime: pm.Duration}
}

// InvoiceFactoryCreator returns a payment engine factory.
func InvoiceFactoryCreator(
	dialog communication.Dialog,
	balanceSendPeriod, promiseTimeout time.Duration,
	invoiceStorage providerInvoiceStorage,
	accountantCaller accountantCaller,
	accountantPromiseStorage accountantPromiseStorage,
	registryAddress string,
	channelImplementationAddress string,
	maxAccountantFailureCount uint64,
	maxAllowedAccountantFee uint16,
	maxRRecovery uint64,
	blockchainHelper bcHelper,
	eventBus eventbus.EventBus,
	feeProvider feeProvider,
	proposal market.ServiceProposal,
	settler settler,
) func(identity.Identity, identity.Identity, string) (session.PaymentEngine, error) {
	return func(providerID identity.Identity, accountantID identity.Identity, sessionID string) (session.PaymentEngine, error) {
		exchangeChan := make(chan crypto.ExchangeMessage, 1)
		listener := NewExchangeListener(exchangeChan)
		invoiceSender := NewInvoiceSender(dialog)
		err := dialog.Receive(listener.GetConsumer())
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoiceTrackerDeps{
			Proposal:                   proposal,
			Peer:                       dialog.PeerID(),
			PeerInvoiceSender:          invoiceSender,
			InvoiceStorage:             invoiceStorage,
			TimeTracker:                &timeTracker,
			ChargePeriod:               balanceSendPeriod,
			ExchangeMessageChan:        exchangeChan,
			ExchangeMessageWaitTimeout: promiseTimeout,
			ProviderID:                 providerID,
			AccountantCaller:           accountantCaller,
			AccountantPromiseStorage:   accountantPromiseStorage,
			AccountantID:               accountantID,
			Registry:                   registryAddress,
			MaxAccountantFailureCount:  maxAccountantFailureCount,
			MaxAllowedAccountantFee:    maxAllowedAccountantFee,
			BlockchainHelper:           blockchainHelper,
			EventBus:                   eventBus,
			FeeProvider:                feeProvider,
			MaxRRecoveryLength:         maxRRecovery,
			Settler:                    settler,
			SessionID:                  sessionID,
			ChannelAddressCalculator:   NewChannelAddressCalculator(accountantID.Address, channelImplementationAddress, registryAddress),
		}
		paymentEngine := NewInvoiceTracker(deps)
		return paymentEngine, nil
	}
}

// ExchangeFactoryFunc returns a backwards compatible version of the exchange factory.
func ExchangeFactoryFunc(
	keystore *keystore.KeyStore,
	options node.Options,
	signer identity.SignerFactory,
	totalStorage consumerTotalsStorage,
	channelImplementation string,
	registryAddress string,
	eventBus eventbus.EventBus,
	getConsumerInfo getConsumerInfo) func(paymentInfo session.PaymentInfo,
	dialog communication.Dialog,
	consumer, provider, accountant identity.Identity, proposal market.ServiceProposal, sessionID string) (connection.PaymentIssuer, error) {
	return func(paymentInfo session.PaymentInfo,
		dialog communication.Dialog,
		consumer, provider, accountant identity.Identity, proposal market.ServiceProposal, sessionID string) (connection.PaymentIssuer, error) {

		if paymentInfo.Supports != string(session.PaymentVersionV3) {
			log.Info().Msg("provider requested old payments")
			return nil, errors.New("provider requested old payments")
		}

		log.Info().Msg("Using new payments")
		invoices := make(chan crypto.Invoice)
		listener := NewInvoiceListener(invoices)
		err := dialog.Receive(listener.GetConsumer())
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoicePayerDeps{
			InvoiceChan:               invoices,
			PeerExchangeMessageSender: NewExchangeSender(dialog),
			ConsumerTotalsStorage:     totalStorage,
			TimeTracker:               &timeTracker,
			Ks:                        keystore,
			Identity:                  consumer,
			Peer:                      dialog.PeerID(),
			Proposal:                  proposal,
			ChannelAddressCalculator:  NewChannelAddressCalculator(accountant.Address, channelImplementation, registryAddress),
			EventBus:                  eventBus,
			AccountantAddress:         accountant,
			ConsumerInfoGetter:        getConsumerInfo,
			SessionID:                 sessionID,
		}
		return NewInvoicePayer(deps), nil
	}
}
