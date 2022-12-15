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
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
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

type consumerTotalsStorage interface {
	Store(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error
	Get(chainID int64, id identity.Identity, hermesID common.Address) (*big.Int, error)
	Add(chainID int64, id identity.Identity, hermesID common.Address, amount *big.Int) error
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

	sessionIDLock sync.Mutex
}

type hashSigner interface {
	SignHash(a accounts.Account, hash []byte) ([]byte, error)
}

// InvoicePayerDeps contains all the dependencies for the exchange message tracker.
type InvoicePayerDeps struct {
	InvoiceChan               chan crypto.Invoice
	PeerExchangeMessageSender PeerExchangeMessageSender
	ConsumerTotalsStorage     consumerTotalsStorage
	TimeTracker               timeTracker
	Ks                        hashSigner
	Identity, Peer            identity.Identity
	AgreedPrice               market.Price
	SenderUUID                string
	SessionID                 string
	AddressProvider           addressProvider
	EventBus                  eventbus.EventBus
	HermesAddress             common.Address
	DataLeeway                datasize.BitSize
	ChainID                   int64
}

// NewInvoicePayer returns a new instance of exchange message tracker.
func NewInvoicePayer(ipd InvoicePayerDeps) *InvoicePayer {
	return &InvoicePayer{
		stop: make(chan struct{}),
		deps: ipd,
		lastInvoice: crypto.Invoice{
			AgreementID:    new(big.Int),
			AgreementTotal: new(big.Int),
			TransactorFee:  new(big.Int),
		},
	}
}

// ErrInvoiceMissmatch represents an error that occurs when invoices do not match.
var ErrInvoiceMissmatch = errors.New("invoice mismatch")

// Start starts the message exchange tracker. Blocks.
func (ip *InvoicePayer) Start() error {
	log.Debug().Msg("Starting...")
	addr, err := ip.deps.AddressProvider.GetActiveChannelAddress(ip.deps.ChainID, ip.deps.Identity.ToCommonAddress())
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}
	ip.channelAddress = identity.FromAddress(addr.Hex())

	ip.deps.TimeTracker.StartTracking()

	uid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	err = ip.deps.EventBus.SubscribeWithUID(connectionstate.AppTopicConnectionStatistics, uid.String(), ip.consumeDataTransferredEvent)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to data transfer events")
	}

	for {
		select {
		case <-ip.stop:
			_ = ip.deps.EventBus.UnsubscribeWithUID(connectionstate.AppTopicConnectionStatistics, uid.String(), ip.consumeDataTransferredEvent)

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

func (ip *InvoicePayer) incrementGrandTotalPromised(amount big.Int) error {
	return ip.deps.ConsumerTotalsStorage.Add(ip.chainID(), ip.deps.Identity, ip.deps.HermesAddress, &amount)
}

func (ip *InvoicePayer) isInvoiceOK(invoice crypto.Invoice) error {
	if !strings.EqualFold(invoice.Provider, ip.deps.Peer.Address) {
		return ErrWrongProvider
	}

	transferred := ip.getDataTransferred()
	transferred.Up += ip.deps.DataLeeway.Bytes()

	shouldBe := CalculatePaymentAmount(ip.deps.TimeTracker.Elapsed(), transferred, ip.deps.AgreedPrice)
	estimatedTolerance := estimateInvoiceTolerance(ip.deps.TimeTracker.Elapsed(), transferred)

	upperBound, _ := new(big.Float).Mul(new(big.Float).SetInt(shouldBe), big.NewFloat(estimatedTolerance)).Int(nil)

	log.Debug().Msgf("Estimated tolerance %.4v, upper bound %v", estimatedTolerance, upperBound)

	if invoice.AgreementTotal.Cmp(upperBound) == 1 {
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

func (ip *InvoicePayer) calculateAmountToPromise(invoice crypto.Invoice) (toPromise *big.Int, diff *big.Int, err error) {
	diff = safeSub(invoice.AgreementTotal, ip.lastInvoice.AgreementTotal)
	totalPromised, err := ip.deps.ConsumerTotalsStorage.Get(ip.chainID(), ip.deps.Identity, ip.deps.HermesAddress)
	if err != nil {
		if err != ErrNotFound {
			return new(big.Int), new(big.Int), fmt.Errorf("could not get previous grand total: %w", err)
		}
		log.Debug().Msg("No previous promised total, assuming 0")
		totalPromised = new(big.Int)
	}

	// This is a new agreement, we need to take in the agreement total and just add it to total promised
	if ip.lastInvoice.AgreementID.Cmp(invoice.AgreementID) != 0 {
		diff = invoice.AgreementTotal
	}

	log.Debug().Msgf("Loaded previous state: already promised: %v", totalPromised)
	log.Debug().Msgf("Incrementing promised amount by %v", diff)
	amountToPromise := new(big.Int).Add(totalPromised, diff)
	return amountToPromise, diff, nil
}

func (ip *InvoicePayer) chainID() int64 {
	return ip.deps.ChainID
}

func (ip *InvoicePayer) issueExchangeMessage(invoice crypto.Invoice) error {
	amountToPromise, diff, err := ip.calculateAmountToPromise(invoice)
	if err != nil {
		return errors.Wrap(err, "could not calculate amount to promise")
	}

	msg, err := crypto.CreateExchangeMessage(ip.chainID(), invoice, amountToPromise, ip.channelAddress.Address, ip.deps.HermesAddress.Hex(), ip.deps.Ks, common.HexToAddress(ip.deps.Identity.Address))
	if err != nil {
		return errors.Wrap(err, "could not create exchange message")
	}

	err = ip.deps.PeerExchangeMessageSender.Send(*msg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send exchange message")
	}

	ip.publishInvoicePayedEvent(invoice)

	// TODO: we'd probably want to check if we have enough balance here
	err = ip.incrementGrandTotalPromised(*diff)
	return errors.Wrap(err, "could not increment grand total")
}

func (ip *InvoicePayer) publishInvoicePayedEvent(invoice crypto.Invoice) {
	ip.sessionIDLock.Lock()
	defer ip.sessionIDLock.Unlock()

	// session id might be set later than we start paying invoices, skip in that case.
	if ip.deps.SessionID == "" {
		return
	}

	ip.deps.EventBus.Publish(event.AppTopicInvoicePaid, event.AppEventInvoicePaid{
		UUID:       ip.deps.SenderUUID,
		ConsumerID: ip.deps.Identity,
		SessionID:  ip.deps.SessionID,
		Invoice:    invoice,
	})
}

// Stop stops the message tracker.
func (ip *InvoicePayer) Stop() {
	ip.once.Do(func() {
		log.Debug().Msg("Stopping...")
		close(ip.stop)
	})
}

func (ip *InvoicePayer) consumeDataTransferredEvent(e connectionstate.AppEventConnectionStatistics) {
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

// SetSessionID updates invoice payer dependencies to set session ID once session established.
func (ip *InvoicePayer) SetSessionID(sessionID string) {
	ip.sessionIDLock.Lock()
	defer ip.sessionIDLock.Unlock()
	ip.deps.SessionID = sessionID
}
