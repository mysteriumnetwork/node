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
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const (
	// PromiseWaitTimeout is the time that the provider waits for the promise to arrive
	PromiseWaitTimeout = time.Second * 50

	// InvoiceSendPeriod is how often the provider will send invoice messages to the consumer
	InvoiceSendPeriod = time.Second * 60

	// DefaultHermesFailureCount defines how many times we're allowed to fail to reach hermes in a row before announcing the failure.
	DefaultHermesFailureCount uint64 = 10
)

var giB = big.NewInt(1024 ^ 3)
var accuracy = big.NewInt(500000000000000)

// NewPrice returns the the default payment method of time + bytes.
func NewPrice(perGiB, perHour *big.Int) market.Price {
	if perGiB == nil {
		perGiB = new(big.Int)
	}
	if perHour == nil {
		perHour = new(big.Int)
	}

	if perGiB.Cmp(big.NewInt(0)) > 0 {
		mul := new(big.Int).Mul(giB, accuracy)
		perGiB = new(big.Int).Div(mul, perGiB)
	}
	if perHour.Cmp(big.NewInt(0)) > 0 {
		mul := new(big.Int).Mul(big.NewInt(int64(time.Hour)), accuracy)
		perHour = new(big.Int).Div(mul, perHour)
	}
	return market.Price{
		Currency: money.Currency(config.GetString(config.FlagDefaultCurrency)),
		PerGiB:   perGiB,
		PerHour:  perHour,
	}
}

// InvoiceFactoryCreator returns a payment engine factory.
func InvoiceFactoryCreator(
	channel p2p.Channel,
	balanceSendPeriod, promiseTimeout time.Duration,
	invoiceStorage providerInvoiceStorage,
	maxHermesFailureCount uint64,
	maxAllowedHermesFee uint16,
	maxUnpaidInvoiceValue *big.Int,
	hermesStatusChecker hermesStatusChecker,
	eventBus eventbus.EventBus,
	proposal market.ServiceProposal,
	promiseHandler promiseHandler,
	addressProvider addressProvider,
) func(identity.Identity, identity.Identity, int64, common.Address, string, chan crypto.ExchangeMessage) (service.PaymentEngine, error) {
	return func(providerID, consumerID identity.Identity, chainID int64, hermesID common.Address, sessionID string, exchangeChan chan crypto.ExchangeMessage) (service.PaymentEngine, error) {
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoiceTrackerDeps{
			Proposal:                   proposal,
			Peer:                       consumerID,
			PeerInvoiceSender:          NewInvoiceSender(channel),
			InvoiceStorage:             invoiceStorage,
			TimeTracker:                &timeTracker,
			ChargePeriod:               balanceSendPeriod,
			ChargePeriodLeeway:         2 * time.Minute,
			ExchangeMessageChan:        exchangeChan,
			ExchangeMessageWaitTimeout: promiseTimeout,
			ProviderID:                 providerID,
			ConsumersHermesID:          hermesID,
			MaxHermesFailureCount:      maxHermesFailureCount,
			MaxAllowedHermesFee:        maxAllowedHermesFee,
			HermesStatusChecker:        hermesStatusChecker,
			EventBus:                   eventBus,
			SessionID:                  sessionID,
			PromiseHandler:             promiseHandler,
			MaxNotPaidInvoice:          maxUnpaidInvoiceValue,
			ChainID:                    chainID,
			AddressProvider:            addressProvider,
		}
		paymentEngine := NewInvoiceTracker(deps)
		return paymentEngine, nil
	}
}

// ExchangeFactoryFunc returns a exchange factory.
func ExchangeFactoryFunc(
	keystore hashSigner,
	signer identity.SignerFactory,
	totalStorage consumerTotalsStorage,
	addressProvider addressProvider,
	eventBus eventbus.EventBus,
	dataLeewayMegabytes uint64) func(channel p2p.Channel, consumer, provider identity.Identity, hermes common.Address, proposal market.ServiceProposal) (connection.PaymentIssuer, error) {
	return func(channel p2p.Channel, consumer, provider identity.Identity, hermes common.Address, proposal market.ServiceProposal) (connection.PaymentIssuer, error) {
		invoices, err := invoiceReceiver(channel)
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoicePayerDeps{
			InvoiceChan:               invoices,
			PeerExchangeMessageSender: NewExchangeSender(channel),
			ConsumerTotalsStorage:     totalStorage,
			TimeTracker:               &timeTracker,
			Ks:                        keystore,
			Identity:                  consumer,
			Peer:                      provider,
			Proposal:                  proposal,
			AddressProvider:           addressProvider,
			EventBus:                  eventBus,
			HermesAddress:             hermes,
			DataLeeway:                datasize.MiB * datasize.BitSize(dataLeewayMegabytes),
			ChainID:                   config.GetInt64(config.FlagChainID),
		}
		return NewInvoicePayer(deps), nil
	}
}

func invoiceReceiver(channel p2p.ChannelHandler) (chan crypto.Invoice, error) {
	invoices := make(chan crypto.Invoice)

	channel.Handle(p2p.TopicPaymentInvoice, func(c p2p.Context) error {
		var msg pb.Invoice
		if err := c.Request().UnmarshalProto(&msg); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicPaymentInvoice, msg.String())

		agreementID, ok := new(big.Int).SetString(msg.GetAgreementID(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field agreementID of value %v", agreementID)
		}
		agreementTotal, ok := new(big.Int).SetString(msg.GetAgreementTotal(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field agreementTotal of value %v", agreementTotal)
		}
		transactorFee, ok := new(big.Int).SetString(msg.GetTransactorFee(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field transactorFee of value %v", transactorFee)
		}

		invoices <- crypto.Invoice{
			AgreementID:    agreementID,
			AgreementTotal: agreementTotal,
			TransactorFee:  transactorFee,
			Hashlock:       msg.GetHashlock(),
			Provider:       msg.GetProvider(),
			ChainID:        msg.GetChainID(),
		}

		return nil
	})

	return invoices, nil
}
