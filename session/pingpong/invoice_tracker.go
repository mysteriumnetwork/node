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
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrConsumerPromiseValidationFailed represents an error where consumer tries to cheat us with incorrect promises.
var ErrConsumerPromiseValidationFailed = errors.New("consumer failed to issue promise for the correct amount")

// ErrAccountantFeeTooLarge indicates that we do not allow accountants with such high fees
var ErrAccountantFeeTooLarge = errors.New("accountants fee exceeds")

// PeerInvoiceSender allows to send invoices.
type PeerInvoiceSender interface {
	Send(crypto.Invoice) error
}

type bcHelper interface {
	GetAccountantFee(accountantAddress common.Address) (uint16, error)
	IsRegistered(registryAddress, addressToCheck common.Address) (bool, error)
}

type providerInvoiceStorage interface {
	Get(consumerIdentity identity.Identity) (crypto.Invoice, error)
	Store(consumerIdentity identity.Identity, invoice crypto.Invoice) error
	GetNewAgreementID() (uint64, error)
	StoreR(agreementID uint64, r string) error
}

type accountantPromiseStorage interface {
	Store(accountantID identity.Identity, promise crypto.Promise) error
	Get(accountantID identity.Identity) (crypto.Promise, error)
}

type accountantCaller interface {
	RequestPromise(em crypto.ExchangeMessage) (crypto.Promise, error)
	RevealR(r string, provider string, agreementID uint64) error
}

// ErrExchangeWaitTimeout indicates that we did not get an exchange message in time.
var ErrExchangeWaitTimeout = errors.New("did not get a new exchange message")

// ErrExchangeValidationFailed indicates that there was an error with the exchange signature.
var ErrExchangeValidationFailed = errors.New("exchange validation failed")

// ErrConsumerNotRegistered represents the error that the consumer is not registered
var ErrConsumerNotRegistered = errors.New("consumer not registered")

const chargePeriodLeeway = time.Hour * 2

type lastInvoice struct {
	invoice crypto.Invoice
	r       []byte
}

// InvoiceTracker keeps tab of invoices and sends them to the consumer.
type InvoiceTracker struct {
	peer                            identity.Identity
	stop                            chan struct{}
	peerInvoiceSender               PeerInvoiceSender
	exchangeMessageChan             chan crypto.ExchangeMessage
	chargePeriod                    time.Duration
	exchangeMessageWaitTimeout      time.Duration
	accountantFailureCount          uint64
	notReceivedExchangeMessageCount uint64
	maxNotReceivedExchangeMessages  uint64
	once                            sync.Once
	invoiceStorage                  providerInvoiceStorage
	accountantPromiseStorage        accountantPromiseStorage
	timeTracker                     timeTracker
	paymentInfo                     dto.PaymentRate
	providerID                      identity.Identity
	accountantID                    identity.Identity
	lastInvoice                     lastInvoice
	lastExchangeMessage             crypto.ExchangeMessage
	accountantCaller                accountantCaller
	channelImplementation           string
	registryAddress                 string
	maxAccountantFailureCount       uint64
	maxAllowedAccountantFee         uint16
	bcHelper                        bcHelper
}

// InvoiceTrackerDeps contains all the deps needed for invoice tracker.
type InvoiceTrackerDeps struct {
	Peer                       identity.Identity
	PeerInvoiceSender          PeerInvoiceSender
	InvoiceStorage             providerInvoiceStorage
	TimeTracker                timeTracker
	ChargePeriod               time.Duration
	ExchangeMessageChan        chan crypto.ExchangeMessage
	ExchangeMessageWaitTimeout time.Duration
	PaymentInfo                dto.PaymentRate
	ProviderID                 identity.Identity
	AccountantID               identity.Identity
	AccountantCaller           accountantCaller
	AccountantPromiseStorage   accountantPromiseStorage
	ChannelImplementation      string
	Registry                   string
	MaxAccountantFailureCount  uint64
	MaxAllowedAccountantFee    uint16
	BlockchainHelper           bcHelper
}

// NewInvoiceTracker creates a new instance of invoice tracker.
func NewInvoiceTracker(
	itd InvoiceTrackerDeps) *InvoiceTracker {
	return &InvoiceTracker{
		peer:                           itd.Peer,
		stop:                           make(chan struct{}),
		peerInvoiceSender:              itd.PeerInvoiceSender,
		exchangeMessageChan:            itd.ExchangeMessageChan,
		exchangeMessageWaitTimeout:     itd.ExchangeMessageWaitTimeout,
		chargePeriod:                   itd.ChargePeriod,
		invoiceStorage:                 itd.InvoiceStorage,
		timeTracker:                    itd.TimeTracker,
		paymentInfo:                    itd.PaymentInfo,
		providerID:                     itd.ProviderID,
		accountantCaller:               itd.AccountantCaller,
		accountantPromiseStorage:       itd.AccountantPromiseStorage,
		accountantID:                   itd.AccountantID,
		maxNotReceivedExchangeMessages: calculateMaxNotReceivedExchangeMessageCount(chargePeriodLeeway, itd.ChargePeriod),
		channelImplementation:          itd.ChannelImplementation,
		registryAddress:                itd.Registry,
		maxAccountantFailureCount:      itd.MaxAccountantFailureCount,
		maxAllowedAccountantFee:        itd.MaxAllowedAccountantFee,
		bcHelper:                       itd.BlockchainHelper,
	}
}

func calculateMaxNotReceivedExchangeMessageCount(chargeLeeway, chargePeriod time.Duration) uint64 {
	return uint64(math.Round(float64(chargeLeeway) / float64(chargePeriod)))
}

func (it *InvoiceTracker) generateInitialInvoice() error {
	agreementID, err := it.invoiceStorage.GetNewAgreementID()
	if err != nil {
		return errors.Wrap(err, "could not get new agreement id")
	}
	// TODO: set fee
	r := make([]byte, 64)
	rand.Read(r)
	invoice := crypto.CreateInvoice(agreementID, it.paymentInfo.GetPrice().Amount, 0, r)
	invoice.Provider = it.providerID.Address
	it.lastInvoice = lastInvoice{
		invoice: invoice,
		r:       r,
	}
	return errors.Wrap(it.invoiceStorage.StoreR(agreementID, common.Bytes2Hex(r)), "could not store r")
}

// Start stars the invoice tracker
func (it *InvoiceTracker) Start() error {
	log.Debug().Msg("Starting...")
	it.timeTracker.StartTracking()

	isConsumerRegistered, err := it.bcHelper.IsRegistered(common.HexToAddress(it.registryAddress), it.peer.ToCommonAddress())
	if err != nil {
		return errors.Wrap(err, "could not check customer identity registration status")
	}

	if !isConsumerRegistered {
		return ErrConsumerNotRegistered
	}

	fee, err := it.bcHelper.GetAccountantFee(common.HexToAddress(it.accountantID.Address))
	if err != nil {
		return errors.Wrap(err, "could not get accountants fee")
	}

	if fee > it.maxAllowedAccountantFee {
		log.Error().Msgf("Accountant fee too large, asking for %v where %v is the limit", fee, it.maxAllowedAccountantFee)
		return ErrAccountantFeeTooLarge
	}

	err = it.generateInitialInvoice()
	if err != nil {
		return errors.Wrap(err, "could not generate initial invoice")
	}

	// give the consumer a second to start up his payments before sending the first request
	firstSend := time.After(time.Second)
	for {
		select {
		case <-firstSend:
			err := it.sendInvoiceExpectExchangeMessage()
			if err != nil {
				return err
			}
		case <-it.stop:
			return nil
		case <-time.After(it.chargePeriod):
			err := it.sendInvoiceExpectExchangeMessage()
			if err != nil {
				return err
			}
		}
	}
}

func (it *InvoiceTracker) markExchangeMessageNotReceived() {
	atomic.AddUint64(&it.notReceivedExchangeMessageCount, 1)
}

func (it *InvoiceTracker) resetNotReceivedExchangeMessageCount() {
	atomic.SwapUint64(&it.notReceivedExchangeMessageCount, 0)
}

func (it *InvoiceTracker) getNotReceivedExchangeMessageCount() uint64 {
	return atomic.LoadUint64(&it.notReceivedExchangeMessageCount)
}

func (it *InvoiceTracker) sendInvoiceExpectExchangeMessage() error {
	// TODO: this should be calculated according to the passed in payment period
	shouldBe := uint64(math.Trunc(it.timeTracker.Elapsed().Minutes() * float64(it.paymentInfo.GetPrice().Amount)))

	// In case we're sending a first invoice, there might be a big missmatch percentage wise on the consumer side.
	// This is due to the fact that both payment providers start at different times.
	// To compensate for this, be a bit more lenient on the first invoice - ask for a reduced amount.
	// Over the long run, this becomes redundant as the difference should become miniscule.
	if it.lastExchangeMessage.AgreementTotal == 0 {
		shouldBe = uint64(math.Trunc(float64(shouldBe) * 0.8))
		log.Debug().Msgf("Being lenient for the first payment, asking for %v", shouldBe)
	}

	// TODO: fill in the fee
	invoice := crypto.CreateInvoice(it.lastInvoice.invoice.AgreementID, shouldBe, 0, it.lastInvoice.r)
	invoice.Provider = it.providerID.Address
	err := it.peerInvoiceSender.Send(invoice)
	if err != nil {
		return err
	}

	err = it.invoiceStorage.Store(it.peer, invoice)
	if err != nil {
		return errors.Wrap(err, "could not store invoice")
	}

	err = it.receiveExchangeMessageOrTimeout()
	if err != nil {
		handlerErr := it.handleExchangeMessageReceiveError(err)
		if handlerErr != nil {
			return err
		}
	} else {
		it.resetNotReceivedExchangeMessageCount()
	}
	return nil
}

func (it *InvoiceTracker) handleExchangeMessageReceiveError(err error) error {
	// if it's a timeout, we'll want to ignore it if we're not exceeding maxNotReceivedexchangeMessages
	if err == ErrExchangeWaitTimeout {
		it.markExchangeMessageNotReceived()
		if it.getNotReceivedExchangeMessageCount() >= it.maxNotReceivedExchangeMessages {
			return err
		}
		log.Warn().Err(err).Msg("Failed to receive exchangeMessage")
		return nil
	}
	return err
}

func (it *InvoiceTracker) incrementAccountantFailureCount() {
	atomic.AddUint64(&it.accountantFailureCount, 1)
}

func (it *InvoiceTracker) resetAccountantFailureCount() {
	atomic.SwapUint64(&it.accountantFailureCount, 0)
}

func (it *InvoiceTracker) getAccountantFailureCount() uint64 {
	return atomic.LoadUint64(&it.accountantFailureCount)
}

func (it *InvoiceTracker) validateExchangeMessage(em crypto.ExchangeMessage) error {
	if res := em.IsMessageValid(common.HexToAddress(it.peer.Address)); !res {
		return ErrExchangeValidationFailed
	}

	if em.Promise.Amount < it.lastExchangeMessage.Promise.Amount {
		log.Warn().Msgf("Consumer sent an invalid amount. Expected < %v, got %v", it.lastExchangeMessage.Promise.Amount, em.Promise.Amount)
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid amount")
	}

	hashlock, err := hex.DecodeString(strings.TrimPrefix(it.lastInvoice.invoice.Hashlock, "0x"))
	if err != nil {
		return errors.Wrap(err, "could not decode hashlock")
	}

	if !bytes.Equal(hashlock, em.Promise.Hashlock) {
		log.Warn().Msgf("Consumer sent an invalid hashlock. Expected %q, got %q", it.lastInvoice.invoice.Hashlock, hex.EncodeToString(em.Promise.Hashlock))
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "missmatching hashlock")
	}

	addr, err := crypto.GenerateChannelAddress(it.peer.Address, it.accountantID.Address, it.registryAddress, it.channelImplementation)
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}

	expectedChannel, err := hex.DecodeString(strings.TrimPrefix(addr, "0x"))
	if err != nil {
		return errors.Wrap(err, "could not decode expected chanel")
	}

	if !bytes.Equal(expectedChannel, em.Promise.ChannelID) {
		log.Warn().Msgf("Consumer sent an invalid channel address. Expected %q, got %q", addr, hex.EncodeToString(em.Promise.ChannelID))
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid channel address")
	}
	return nil
}

func (it *InvoiceTracker) receiveExchangeMessageOrTimeout() error {
	select {
	case pm := <-it.exchangeMessageChan:
		err := it.validateExchangeMessage(pm)
		if err != nil {
			return err
		}

		it.lastExchangeMessage = pm

		promise, err := it.accountantCaller.RequestPromise(pm)
		if err != nil {
			log.Warn().Err(err).Msg("Could not call accountant")
			it.incrementAccountantFailureCount()
			if it.getAccountantFailureCount() > it.maxAccountantFailureCount {
				return errors.Wrap(err, "could not call accountant")
			}
			log.Warn().Msg("Ignoring accountant error, we haven't reached the error threshold yet")
			return nil
		}
		it.resetAccountantFailureCount()
		err = it.accountantPromiseStorage.Store(it.accountantID, promise)
		if err != nil {
			return errors.Wrap(err, "could not store accountant promise")
		}
		log.Debug().Msg("Accountant promise stored")
		hexR := hex.EncodeToString(it.lastInvoice.r)
		err = it.accountantCaller.RevealR(hexR, it.providerID.Address, it.lastInvoice.invoice.AgreementID)
		if err != nil {
			// TODO: need to think about handling this a bit better
			log.Error().Err(err).Msg("could not reveal R")
		}
	case <-time.After(it.exchangeMessageWaitTimeout):
		return ErrExchangeWaitTimeout
	case <-it.stop:
		return nil
	}
	return nil
}

// Stop stops the invoice tracker.
func (it *InvoiceTracker) Stop() {
	it.once.Do(func() {
		log.Debug().Msg("Stopping...")
		close(it.stop)
	})
}
