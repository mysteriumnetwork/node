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
	"math"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrWrongProvider represents an issue where the wrong provider is supplied.
var ErrWrongProvider = errors.New("wrong provider supplied")

// ErrProviderOvercharge represents an issue where the provider is trying to overcharge us.
var ErrProviderOvercharge = errors.New("provider is overcharging")

// PeerExchangeMessageSender allows for sending of exchange messages.
type PeerExchangeMessageSender interface {
	Send(crypto.ExchangeMessage) error
}

type consumerInvoiceStorage interface {
	Get(consumerIdentity, providerIdentity identity.Identity) (crypto.Invoice, error)
	Store(consumerIdentity, providerIdentity identity.Identity, invoice crypto.Invoice) error
}

type consumerTotalsStorage interface {
	Store(providerAddress string, amount uint64) error
	Get(providerAddress string) (uint64, error)
}

type timeTracker interface {
	StartTracking()
	Elapsed() time.Duration
}

type channelAddressCalculator interface {
	GetChannelAddress(id identity.Identity) (common.Address, error)
}

// ExchangeMessageTracker keeps track of exchange messages and sends them to the provider.
type ExchangeMessageTracker struct {
	stop                      chan struct{}
	invoiceChan               chan crypto.Invoice
	peerExchangeMessageSender PeerExchangeMessageSender
	once                      sync.Once
	keystore                  *keystore.KeyStore
	identity                  identity.Identity
	peer                      identity.Identity
	channelAddress            identity.Identity

	consumerInvoiceStorage   consumerInvoiceStorage
	consumerTotalsStorage    consumerTotalsStorage
	timeTracker              timeTracker
	paymentInfo              dto.PaymentRate
	channelAddressCalculator channelAddressCalculator
	publisher                eventbus.Publisher
}

// ExchangeMessageTrackerDeps contains all the dependencies for the exchange message tracker.
type ExchangeMessageTrackerDeps struct {
	InvoiceChan               chan crypto.Invoice
	PeerExchangeMessageSender PeerExchangeMessageSender
	ConsumerInvoiceStorage    consumerInvoiceStorage
	ConsumerTotalsStorage     consumerTotalsStorage
	TimeTracker               timeTracker
	Ks                        *keystore.KeyStore
	Identity, Peer            identity.Identity
	PaymentInfo               dto.PaymentRate
	ChannelAddressCalculator  channelAddressCalculator
	Publisher                 eventbus.Publisher
}

// NewExchangeMessageTracker returns a new instance of exchange message tracker.
func NewExchangeMessageTracker(emtd ExchangeMessageTrackerDeps) *ExchangeMessageTracker {
	return &ExchangeMessageTracker{
		stop:                      make(chan struct{}),
		peerExchangeMessageSender: emtd.PeerExchangeMessageSender,
		invoiceChan:               emtd.InvoiceChan,
		keystore:                  emtd.Ks,
		consumerInvoiceStorage:    emtd.ConsumerInvoiceStorage,
		consumerTotalsStorage:     emtd.ConsumerTotalsStorage,
		identity:                  emtd.Identity,
		timeTracker:               emtd.TimeTracker,
		peer:                      emtd.Peer,
		paymentInfo:               emtd.PaymentInfo,
		channelAddressCalculator:  emtd.ChannelAddressCalculator,
		publisher:                 emtd.Publisher,
	}
}

// ErrInvoiceMissmatch represents an error that occurs when invoices do not match.
var ErrInvoiceMissmatch = errors.New("invoice mismatch")

// Start starts the message exchange tracker. Blocks.
func (emt *ExchangeMessageTracker) Start() error {
	log.Debug().Msg("Starting...")
	addr, err := emt.channelAddressCalculator.GetChannelAddress(emt.identity)
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}
	emt.channelAddress = identity.FromAddress(addr.Hex())

	emt.timeTracker.StartTracking()

	for {
		select {
		case <-emt.stop:
			return nil
		case invoice := <-emt.invoiceChan:
			log.Debug().Msgf("Invoice received: %v", invoice)
			err := emt.isInvoiceOK(invoice)
			if err != nil {
				return errors.Wrap(err, "invoice not valid")
			}

			err = emt.issueExchangeMessage(invoice)
			if err != nil {
				return err
			}

			err = emt.consumerInvoiceStorage.Store(emt.identity, emt.peer, invoice)
			if err != nil {
				return errors.Wrap(err, "could not store invoice")
			}

		}
	}
}

const grandTotalKey = "consumer_grand_total"

func (emt *ExchangeMessageTracker) getGrandTotalPromised() (uint64, error) {
	res, err := emt.consumerTotalsStorage.Get(fmt.Sprintf("%v_%v", grandTotalKey, emt.identity.Address))
	if err != nil {
		if err == ErrNotFound {
			log.Debug().Msgf("No previous invoice grand total, assuming zero")
			return 0, nil
		}
		return 0, errors.Wrap(err, "could not get previous grand total")
	}
	return res, nil
}

func (emt *ExchangeMessageTracker) incrementGrandTotalPromised(amount uint64) error {
	k := fmt.Sprintf("%v_%v", grandTotalKey, emt.identity.Address)
	res, err := emt.consumerTotalsStorage.Get(k)
	if err != nil {
		if err == ErrNotFound {
			log.Debug().Msg("No previous invoice grand total, assuming zero")
		} else {
			return errors.Wrap(err, "could not get previous grand total")
		}
	}
	return emt.consumerTotalsStorage.Store(k, res+amount)
}

func (emt *ExchangeMessageTracker) isInvoiceOK(invoice crypto.Invoice) error {
	if strings.ToLower(invoice.Provider) != strings.ToLower(emt.peer.Address) {
		return ErrWrongProvider
	}

	// TODO: this should be calculated according to the passed in payment period, not a hardcoded minute
	shouldBe := uint64(math.Trunc(emt.timeTracker.Elapsed().Minutes() * float64(emt.paymentInfo.GetPrice().Amount)))
	upperBound := uint64(math.Trunc(float64(shouldBe) * 1.05))
	log.Debug().Msgf("Upper bound %v", upperBound)

	if invoice.AgreementTotal > upperBound {
		log.Warn().Msg("Provider trying to overcharge")
		return ErrProviderOvercharge
	}

	return nil
}

func (emt *ExchangeMessageTracker) calculateAmountToPromise(invoice crypto.Invoice) (toPromise uint64, diff uint64, err error) {
	previous, err := emt.consumerInvoiceStorage.Get(emt.identity, emt.peer)
	if err != nil {
		if err == ErrNotFound {
			// do nothing, really
			log.Debug().Msg("No previous invoice found, assuming zero")
		} else {
			return 0, 0, errors.Wrap(err, fmt.Sprintf("could not get previous total for peer %q", invoice.Provider))
		}
	}

	diff = invoice.AgreementTotal - previous.AgreementTotal
	totalPromised, err := emt.getGrandTotalPromised()
	if err != nil {
		return 0, 0, err
	}

	// This is a new agreement, we need to take in the agreement total and just add it to total promised
	if previous.AgreementID != invoice.AgreementID {
		diff = invoice.AgreementTotal
	}

	log.Debug().Msgf("Loaded previous state: already promised: %v", totalPromised)
	log.Debug().Msgf("Incrementing promised amount by %v", diff)
	amountToPromise := totalPromised + diff
	return amountToPromise, diff, nil
}

func (emt *ExchangeMessageTracker) issueExchangeMessage(invoice crypto.Invoice) error {
	amountToPromise, diff, err := emt.calculateAmountToPromise(invoice)
	if err != nil {
		return errors.Wrap(err, "could not calculate amount to promise")
	}

	msg, err := crypto.CreateExchangeMessage(invoice, amountToPromise, emt.channelAddress.Address, emt.keystore, common.HexToAddress(emt.identity.Address))
	if err != nil {
		return errors.Wrap(err, "could not create exchange message")
	}

	err = emt.peerExchangeMessageSender.Send(*msg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send exchange message")
	}

	defer emt.publisher.Publish(ExchangeMessageTopic, ExchangeMessageEventPayload{
		Identity:       emt.identity,
		AmountPromised: amountToPromise,
	})

	// TODO: we'd probably want to check if we have enough balance here
	err = emt.incrementGrandTotalPromised(diff)
	if err != nil {
		return errors.Wrap(err, "could not increment grand total")
	}

	return emt.consumerTotalsStorage.Store(emt.peer.Address, invoice.AgreementTotal)
}

// Stop stops the message tracker.
func (emt *ExchangeMessageTracker) Stop() {
	emt.once.Do(func() {
		log.Debug().Msg("Stopping...")
		close(emt.stop)
	})
}
