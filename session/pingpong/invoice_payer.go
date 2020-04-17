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

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session/pingpong/event"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrWrongProvider represents an issue where the wrong provider is supplied.
var ErrWrongProvider = errors.New("wrong provider supplied")

// ErrProviderOvercharge represents an issue where the provider is trying to overcharge us.
var ErrProviderOvercharge = errors.New("provider is overcharging")

// consumerInvoiceBasicTolerance provider traffic amount compensation due to:
//   - different MTU sizes
//   - measurement timing inaccuracies
//   - possible in-transit packet fragmentation
//   - non-agreed traffic: traffic blocked / dropped / not reachable / failed retransmits on provider
const consumerInvoiceBasicTolerance = 1.11

// PeerExchangeMessageSender allows for sending of exchange messages.
type PeerExchangeMessageSender interface {
	Send(crypto.ExchangeMessage) error
}

type consumerInvoiceStorage interface {
	Get(consumerIdentity, providerIdentity identity.Identity) (crypto.Invoice, error)
	Store(consumerIdentity, providerIdentity identity.Identity, invoice crypto.Invoice) error
}

type consumerTotalsStorage interface {
	Store(id identity.Identity, accountantID common.Address, amount uint64) error
	Get(id identity.Identity, accountantID common.Address) (uint64, error)
}

type timeTracker interface {
	StartTracking()
	Elapsed() time.Duration
}

type channelAddressCalculator interface {
	GetChannelAddress(id identity.Identity) (common.Address, error)
}

// InvoicePayer keeps track of exchange messages and sends them to the provider.
type InvoicePayer struct {
	stop           chan struct{}
	once           sync.Once
	channelAddress identity.Identity

	lastInvoice crypto.Invoice
	deps        InvoicePayerDeps

	dataTransferred     DataTransferred
	dataTransferredLock sync.Mutex
}

// InvoicePayerDeps contains all the dependencies for the exchange message tracker.
type InvoicePayerDeps struct {
	InvoiceChan               chan crypto.Invoice
	PeerExchangeMessageSender PeerExchangeMessageSender
	ConsumerTotalsStorage     consumerTotalsStorage
	TimeTracker               timeTracker
	Ks                        *identity.Keystore
	Identity, Peer            identity.Identity
	Proposal                  market.ServiceProposal
	SessionID                 string
	ChannelAddressCalculator  channelAddressCalculator
	EventBus                  eventbus.EventBus
	AccountantAddress         common.Address
	DataLeeway                datasize.BitSize
}

// NewInvoicePayer returns a new instance of exchange message tracker.
func NewInvoicePayer(ipd InvoicePayerDeps) *InvoicePayer {
	return &InvoicePayer{
		stop:        make(chan struct{}),
		deps:        ipd,
		lastInvoice: crypto.Invoice{},
	}
}

// ErrInvoiceMissmatch represents an error that occurs when invoices do not match.
var ErrInvoiceMissmatch = errors.New("invoice mismatch")

// Start starts the message exchange tracker. Blocks.
func (ip *InvoicePayer) Start() error {
	log.Debug().Msg("Starting...")
	addr, err := ip.deps.ChannelAddressCalculator.GetChannelAddress(ip.deps.Identity)
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}
	ip.channelAddress = identity.FromAddress(addr.Hex())

	ip.deps.TimeTracker.StartTracking()

	err = ip.deps.EventBus.Subscribe(connection.AppTopicConnectionStatistics, ip.consumeDataTransferredEvent)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to data transfer events")
	}

	for {
		select {
		case <-ip.stop:
			return nil
		case invoice := <-ip.deps.InvoiceChan:
			log.Debug().Msgf("Invoice received: %v", invoice)
			err := ip.isInvoiceOK(invoice)
			if err != nil {
				return errors.Wrap(err, "invoice not valid")
			}

			err = ip.issueExchangeMessage(invoice)
			if err != nil {
				return err
			}

			ip.lastInvoice = invoice
		}
	}
}

func (ip *InvoicePayer) incrementGrandTotalPromised(amount uint64) error {
	res, err := ip.deps.ConsumerTotalsStorage.Get(ip.deps.Identity, ip.deps.AccountantAddress)
	if err != nil {
		if err == ErrNotFound {
			log.Debug().Msg("No previous invoice grand total, assuming zero")
		} else {
			return errors.Wrap(err, "could not get previous grand total")
		}
	}
	return ip.deps.ConsumerTotalsStorage.Store(ip.deps.Identity, ip.deps.AccountantAddress, res+amount)
}

func (ip *InvoicePayer) isInvoiceOK(invoice crypto.Invoice) error {
	if !strings.EqualFold(invoice.Provider, ip.deps.Peer.Address) {
		return ErrWrongProvider
	}

	transferred := ip.getDataTransferred()
	transferred.Up += ip.deps.DataLeeway.Bytes()

	shouldBe := CalculatePaymentAmount(ip.deps.TimeTracker.Elapsed(), transferred, ip.deps.Proposal.PaymentMethod)
	estimatedTolerance := estimateInvoiceTolerance(ip.deps.TimeTracker.Elapsed(), transferred)

	upperBound := uint64(math.Trunc(float64(shouldBe) * estimatedTolerance))

	log.Debug().Msgf("Estimated tolerance %.4v, upper bound %v", estimatedTolerance, upperBound)

	if invoice.AgreementTotal > upperBound {
		log.Warn().Msg("Provider trying to overcharge")
		return ErrProviderOvercharge
	}

	return nil
}

func estimateInvoiceTolerance(elapsed time.Duration, transferred DataTransferred) float64 {
	if elapsed.Seconds() < 1 {
		return 3
	}

	totalMiBytesTransferred := float64(transferred.sum()) / (1024 * 1024)
	avgSpeedInMiBits := totalMiBytesTransferred / elapsed.Seconds() * 8

	// correction calculation based on total session duration.
	durInMinutes := elapsed.Minutes()

	if elapsed.Minutes() < 1 {
		durInMinutes = 1
	}

	durationComponent := 1 - durInMinutes/(1+durInMinutes)

	// correction calculation based on average session speed.
	if avgSpeedInMiBits == 0 {
		avgSpeedInMiBits = 1
	}

	avgSpeedComponent := 1 - 1/(1+avgSpeedInMiBits/1024)

	return durationComponent + avgSpeedComponent + consumerInvoiceBasicTolerance
}

func (ip *InvoicePayer) calculateAmountToPromise(invoice crypto.Invoice) (toPromise uint64, diff uint64, err error) {
	diff = invoice.AgreementTotal - ip.lastInvoice.AgreementTotal
	totalPromised, err := ip.deps.ConsumerTotalsStorage.Get(ip.deps.Identity, ip.deps.AccountantAddress)
	if err != nil {
		return 0, 0, fmt.Errorf("could not get previous grand total: %w", err)
	}

	// This is a new agreement, we need to take in the agreement total and just add it to total promised
	if ip.lastInvoice.AgreementID != invoice.AgreementID {
		diff = invoice.AgreementTotal
	}

	log.Debug().Msgf("Loaded previous state: already promised: %v", totalPromised)
	log.Debug().Msgf("Incrementing promised amount by %v", diff)
	amountToPromise := totalPromised + diff
	return amountToPromise, diff, nil
}

func (ip *InvoicePayer) issueExchangeMessage(invoice crypto.Invoice) error {
	amountToPromise, diff, err := ip.calculateAmountToPromise(invoice)
	if err != nil {
		return errors.Wrap(err, "could not calculate amount to promise")
	}

	msg, err := crypto.CreateExchangeMessage(invoice, amountToPromise, ip.channelAddress.Address, ip.deps.Ks, common.HexToAddress(ip.deps.Identity.Address))
	if err != nil {
		return errors.Wrap(err, "could not create exchange message")
	}

	err = ip.deps.PeerExchangeMessageSender.Send(*msg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send exchange message")
	}

	ip.deps.EventBus.Publish(event.AppTopicInvoicePaid, event.AppEventInvoicePaid{
		ConsumerID: ip.deps.Identity,
		SessionID:  ip.deps.SessionID,
		Invoice:    invoice,
	})

	// TODO: we'd probably want to check if we have enough balance here
	err = ip.incrementGrandTotalPromised(diff)
	return errors.Wrap(err, "could not increment grand total")
}

// Stop stops the message tracker.
func (ip *InvoicePayer) Stop() {
	ip.once.Do(func() {
		log.Debug().Msg("Stopping...")
		_ = ip.deps.EventBus.Unsubscribe(connection.AppTopicConnectionStatistics, ip.consumeDataTransferredEvent)
		close(ip.stop)
	})
}

func (ip *InvoicePayer) consumeDataTransferredEvent(e connection.AppEventConnectionStatistics) {
	// skip irrelevant sessions
	if !strings.EqualFold(string(e.SessionInfo.SessionID), ip.deps.SessionID) {
		return
	}

	// From a server perspective, bytes up are the actual bytes the client downloaded(aka the bytes we pushed to the consumer)
	// To lessen the confusion, I suggest having the bytes reversed on the session instance.
	// This way, the session will show that it downloaded the bytes in a manner that is easier to comprehend.
	ip.updateDataTransfer(e.Stats.BytesSent, e.Stats.BytesReceived)
}

func (ip *InvoicePayer) updateDataTransfer(up, down uint64) {
	ip.dataTransferredLock.Lock()
	defer ip.dataTransferredLock.Unlock()

	newUp := ip.dataTransferred.Up
	if up > ip.dataTransferred.Up {
		newUp = up
	}

	newDown := ip.dataTransferred.Down
	if down > ip.dataTransferred.Down {
		newDown = down
	}

	ip.dataTransferred = DataTransferred{
		Up:   newUp,
		Down: newDown,
	}
}

func (ip *InvoicePayer) getDataTransferred() DataTransferred {
	ip.dataTransferredLock.Lock()
	defer ip.dataTransferredLock.Unlock()

	return ip.dataTransferred
}
