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
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrConsumerPromiseValidationFailed represents an error where consumer tries to cheat us with incorrect promises.
var ErrConsumerPromiseValidationFailed = errors.New("consumer failed to issue promise for the correct amount")

// ErrAccountantFeeTooLarge indicates that we do not allow accountants with such high fees
var ErrAccountantFeeTooLarge = errors.New("accountants fee exceeds")

// ErrInvoiceExpired shows that the given invoice has already expired
var ErrInvoiceExpired = errors.New("invoice expired")

// ErrExchangeWaitTimeout indicates that we did not get an exchange message in time.
var ErrExchangeWaitTimeout = errors.New("did not get a new exchange message")

// ErrExchangeValidationFailed indicates that there was an error with the exchange signature.
var ErrExchangeValidationFailed = errors.New("exchange validation failed")

// ErrConsumerNotRegistered represents the error that the consumer is not registered
var ErrConsumerNotRegistered = errors.New("consumer not registered")

// PeerInvoiceSender allows to send invoices.
type PeerInvoiceSender interface {
	Send(crypto.Invoice) error
}

type feeProvider interface {
	FetchSettleFees() (registry.FeesResponse, error)
}

type bcHelper interface {
	GetAccountantFee(accountantAddress common.Address) (uint16, error)
	IsRegistered(registryAddress, addressToCheck common.Address) (bool, error)
}

type providerInvoiceStorage interface {
	Get(providerIdentity, consumerIdentity identity.Identity) (crypto.Invoice, error)
	Store(providerIdentity, consumerIdentity identity.Identity, invoice crypto.Invoice) error
	GetNewAgreementID(providerIdentity identity.Identity) (uint64, error)
	StoreR(providerIdentity identity.Identity, agreementID uint64, r string) error
	GetR(providerID identity.Identity, agreementID uint64) (string, error)
}

type accountantPromiseStorage interface {
	Store(providerID, accountantID identity.Identity, promise AccountantPromise) error
	Get(providerID, accountantID identity.Identity) (AccountantPromise, error)
}

type accountantCaller interface {
	RequestPromise(em crypto.ExchangeMessage) (crypto.Promise, error)
	RevealR(r string, provider string, agreementID uint64) error
}

const chargePeriodLeeway = time.Hour * 2

type sentInvoice struct {
	invoice crypto.Invoice
	r       []byte
}

// InvoiceTracker keeps tab of invoices and sends them to the consumer.
type InvoiceTracker struct {
	stop                            chan struct{}
	accountantFailureCount          uint64
	notReceivedExchangeMessageCount uint64
	maxNotReceivedExchangeMessages  uint64
	once                            sync.Once
	agreementID                     uint64
	lastExchangeMessage             crypto.ExchangeMessage
	transactorFee                   uint64
	invoicesSent                    map[string]sentInvoice
	invoiceLock                     sync.Mutex
	deps                            InvoiceTrackerDeps
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
	Registry                   string
	MaxAccountantFailureCount  uint64
	MaxRRecoveryLength         uint64
	MaxAllowedAccountantFee    uint16
	BlockchainHelper           bcHelper
	Publisher                  eventbus.Publisher
	FeeProvider                feeProvider
	ChannelAddressCalculator   channelAddressCalculator
}

// NewInvoiceTracker creates a new instance of invoice tracker.
func NewInvoiceTracker(
	itd InvoiceTrackerDeps) *InvoiceTracker {
	return &InvoiceTracker{
		stop:                           make(chan struct{}),
		deps:                           itd,
		maxNotReceivedExchangeMessages: calculateMaxNotReceivedExchangeMessageCount(chargePeriodLeeway, itd.ChargePeriod),
		invoicesSent:                   make(map[string]sentInvoice),
	}
}

func calculateMaxNotReceivedExchangeMessageCount(chargeLeeway, chargePeriod time.Duration) uint64 {
	return uint64(math.Round(float64(chargeLeeway) / float64(chargePeriod)))
}

func (it *InvoiceTracker) markInvoiceSent(invoice sentInvoice) {
	it.invoiceLock.Lock()
	defer it.invoiceLock.Unlock()

	it.invoicesSent[invoice.invoice.Hashlock] = invoice
}

func (it *InvoiceTracker) markInvoicePaid(hashlock []byte) {
	it.invoiceLock.Lock()
	defer it.invoiceLock.Unlock()

	delete(it.invoicesSent, hex.EncodeToString(hashlock))
}

func (it *InvoiceTracker) getMarkedInvoice(hashlock []byte) (invoice sentInvoice, ok bool) {
	it.invoiceLock.Lock()
	defer it.invoiceLock.Unlock()
	in, ok := it.invoicesSent[hex.EncodeToString(hashlock)]
	return in, ok
}

func (it *InvoiceTracker) listenForExchangeMessages() error {
	for {
		select {
		case pm := <-it.deps.ExchangeMessageChan:
			err := it.handleExchangeMessage(pm)
			if err != nil && err != ErrInvoiceExpired {
				return err
			}
		case <-it.stop:
			return nil
		}
	}
}

func (it *InvoiceTracker) handleExchangeMessage(pm crypto.ExchangeMessage) error {
	invoice, ok := it.getMarkedInvoice(pm.Promise.Hashlock)
	if !ok {
		log.Debug().Msgf("consumer sent exchange message with missing expired hashlock %s, skipping", invoice.invoice.Hashlock)
		return ErrInvoiceExpired
	}

	err := it.validateExchangeMessage(pm)
	if err != nil {
		return err
	}

	it.lastExchangeMessage = pm
	it.markInvoicePaid(pm.Promise.Hashlock)
	it.resetNotReceivedExchangeMessageCount()

	err = it.revealPromise()
	switch err {
	case errHandled:
		return nil
	case nil:
		break
	default:
		return err
	}

	err = it.deps.InvoiceStorage.StoreR(it.deps.ProviderID, it.agreementID, hex.EncodeToString(invoice.r))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not store r: %s", hex.EncodeToString(invoice.r)))
	}

	err = it.requestPromise(invoice.r, pm)
	switch err {
	case errHandled:
		return nil
	default:
		return err
	}
}

var errHandled = errors.New("error handled, please skip")

func (it *InvoiceTracker) requestPromise(r []byte, pm crypto.ExchangeMessage) error {
	promise, err := it.deps.AccountantCaller.RequestPromise(pm)
	if err != nil {
		log.Warn().Err(err).Msg("Could not call accountant")

		// TODO: handle this better
		if strings.Contains(err.Error(), "400 Bad Request") {
			recoveryError := it.initiateRRecovery()
			if recoveryError != nil {
				return errors.Wrap(err, "could not recover R")
			}
		}

		it.incrementAccountantFailureCount()
		if it.getAccountantFailureCount() > it.deps.MaxAccountantFailureCount {
			return errors.Wrap(err, "could not call accountant")
		}
		log.Warn().Msg("Ignoring accountant error, we haven't reached the error threshold yet")
		return errHandled
	}
	it.resetAccountantFailureCount()

	ap := AccountantPromise{
		Promise:     promise,
		R:           hex.EncodeToString(r),
		Revealed:    false,
		AgreementID: it.agreementID,
	}
	err = it.deps.AccountantPromiseStorage.Store(it.deps.ProviderID, it.deps.AccountantID, ap)
	if err != nil {
		return errors.Wrap(err, "could not store accountant promise")
	}
	log.Debug().Msg("Accountant promise stored")

	promise.R = r
	it.deps.Publisher.Publish(AccountantPromiseTopic, AccountantPromiseEventPayload{
		Promise:      promise,
		AccountantID: it.deps.AccountantID,
		ProviderID:   it.deps.ProviderID,
	})
	return nil
}

func (it *InvoiceTracker) revealPromise() error {
	needsRevealing := false
	accountantPromise, err := it.deps.AccountantPromiseStorage.Get(it.deps.ProviderID, it.deps.AccountantID)
	switch err {
	case nil:
		needsRevealing = !accountantPromise.Revealed
	case ErrNotFound:
		needsRevealing = false
	default:
		return errors.Wrap(err, "could not get accountant promise")
	}

	if !needsRevealing {
		return nil
	}

	err = it.deps.AccountantCaller.RevealR(accountantPromise.R, it.deps.ProviderID.Address, accountantPromise.AgreementID)
	if err != nil {
		log.Error().Err(err).Msg("Could not reveal R")
		it.incrementAccountantFailureCount()
		if it.getAccountantFailureCount() > it.deps.MaxAccountantFailureCount {
			return errors.Wrap(err, "max failure count calling accountant reached")
		}
		log.Warn().Msg("Ignoring accountant error, we haven't reached the error threshold yet")
		return errHandled
	}

	it.resetAccountantFailureCount()
	accountantPromise.Revealed = true
	err = it.deps.AccountantPromiseStorage.Store(it.deps.ProviderID, it.deps.AccountantID, accountantPromise)
	if err != nil {
		return errors.Wrap(err, "could not store accountant promise")
	}
	log.Debug().Msg("Accountant promise stored")

	return nil
}

// Start stars the invoice tracker
func (it *InvoiceTracker) Start() error {
	log.Debug().Msg("Starting...")
	it.deps.TimeTracker.StartTracking()

	isConsumerRegistered, err := it.deps.BlockchainHelper.IsRegistered(common.HexToAddress(it.deps.Registry), it.deps.Peer.ToCommonAddress())
	if err != nil {
		return errors.Wrap(err, "could not check customer identity registration status")
	}

	if !isConsumerRegistered {
		return ErrConsumerNotRegistered
	}

	fees, err := it.deps.FeeProvider.FetchSettleFees()
	if err != nil {
		return errors.Wrap(err, "could not fetch settlement fees")
	}
	it.transactorFee = fees.Fee

	fee, err := it.deps.BlockchainHelper.GetAccountantFee(common.HexToAddress(it.deps.AccountantID.Address))
	if err != nil {
		return errors.Wrap(err, "could not get accountants fee")
	}

	if fee > it.deps.MaxAllowedAccountantFee {
		log.Error().Msgf("Accountant fee too large, asking for %v where %v is the limit", fee, it.deps.MaxAllowedAccountantFee)
		return ErrAccountantFeeTooLarge
	}

	agreementID, err := it.deps.InvoiceStorage.GetNewAgreementID(it.deps.ProviderID)
	if err != nil {
		return errors.Wrap(err, "could not get new agreement id")
	}

	it.agreementID = agreementID

	emErrors := make(chan error)
	go func() {
		emErrors <- it.listenForExchangeMessages()
	}()

	// give the consumer a second to start up his payments before sending the first request
	firstSend := time.After(time.Second)
	for {
		select {
		case <-firstSend:
			err := it.sendInvoice()
			if err != nil {
				return errors.Wrap(err, "sending first invoice failed")
			}
		case <-it.stop:
			return nil
		case <-time.After(it.deps.ChargePeriod):
			err := it.sendInvoice()
			if err != nil {
				return errors.Wrap(err, "sending of invoice failed")
			}
		case emErr := <-emErrors:
			if emErr != nil {
				return errors.Wrap(emErr, "failed to get exchange message")
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

func (it *InvoiceTracker) generateR() []byte {
	r := make([]byte, 32)
	rand.Read(r)
	return r
}

func (it *InvoiceTracker) sendInvoice() error {
	if it.getNotReceivedExchangeMessageCount() >= it.maxNotReceivedExchangeMessages {
		return ErrExchangeWaitTimeout
	}

	// TODO: this should be calculated according to the passed in payment period
	shouldBe := uint64(math.Trunc(it.deps.TimeTracker.Elapsed().Minutes() * float64(it.deps.PaymentInfo.GetPrice().Amount)))

	// In case we're sending a first invoice, there might be a big missmatch percentage wise on the consumer side.
	// This is due to the fact that both payment providers start at different times.
	// To compensate for this, be a bit more lenient on the first invoice - ask for a reduced amount.
	// Over the long run, this becomes redundant as the difference should become miniscule.
	if it.lastExchangeMessage.AgreementTotal == 0 {
		shouldBe = uint64(math.Trunc(float64(shouldBe) * 0.8))
		log.Debug().Msgf("Being lenient for the first payment, asking for %v", shouldBe)
	}

	r := it.generateR()
	invoice := crypto.CreateInvoice(it.agreementID, shouldBe, it.transactorFee, r)
	invoice.Provider = it.deps.ProviderID.Address
	err := it.deps.PeerInvoiceSender.Send(invoice)
	if err != nil {
		return err
	}

	it.markInvoiceSent(sentInvoice{
		invoice: invoice,
		r:       r,
	})

	hlock, err := hex.DecodeString(invoice.Hashlock)
	if err != nil {
		return err
	}

	go it.waitForInvoicePayment(hlock)

	err = it.deps.InvoiceStorage.Store(it.deps.ProviderID, it.deps.Peer, invoice)
	return errors.Wrap(err, "could not store invoice")
}

func (it *InvoiceTracker) waitForInvoicePayment(hlock []byte) {
	select {
	case <-time.After(it.deps.ExchangeMessageWaitTimeout):
		inv, ok := it.getMarkedInvoice(hlock)
		if ok {
			log.Info().Msgf("did not get paid for invoice with hashlock %v, incrementing failure count", inv.invoice.Hashlock)
			it.markInvoicePaid(hlock)
			it.markExchangeMessageNotReceived()
		}
	case <-it.stop:
		return
	}
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
	peerAddr := common.HexToAddress(it.deps.Peer.Address)
	if res := em.IsMessageValid(peerAddr); !res {
		return ErrExchangeValidationFailed
	}

	signer, err := em.Promise.RecoverSigner()
	if err != nil {
		return errors.Wrap(err, "could not recover promise signature")
	}

	if signer.Hex() != peerAddr.Hex() {
		return errors.New("identity missmatch")
	}

	if em.Promise.Amount < it.lastExchangeMessage.Promise.Amount {
		log.Warn().Msgf("Consumer sent an invalid amount. Expected < %v, got %v", it.lastExchangeMessage.Promise.Amount, em.Promise.Amount)
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid amount")
	}

	addr, err := it.deps.ChannelAddressCalculator.GetChannelAddress(it.deps.Peer)
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}

	expectedChannel, err := hex.DecodeString(strings.TrimPrefix(addr.Hex(), "0x"))
	if err != nil {
		return errors.Wrap(err, "could not decode expected chanel")
	}

	if !bytes.Equal(expectedChannel, em.Promise.ChannelID) {
		log.Warn().Msgf("Consumer sent an invalid channel address. Expected %q, got %q", addr, hex.EncodeToString(em.Promise.ChannelID))
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid channel address")
	}
	return nil
}

func (it *InvoiceTracker) initiateRRecovery() error {
	currentAgreement := it.agreementID

	var minBound uint64 = 1
	if currentAgreement > it.deps.MaxRRecoveryLength {
		minBound = currentAgreement - it.deps.MaxRRecoveryLength
	}

	for i := currentAgreement; i >= minBound; i-- {
		r, err := it.deps.InvoiceStorage.GetR(it.deps.ProviderID, i)
		if err != nil {
			return errors.Wrap(err, "could not get R")
		}
		err = it.deps.AccountantCaller.RevealR(r, it.deps.ProviderID.Address, i)
		if err != nil {
			log.Warn().Err(err).Msgf("revealing %v", i)
		} else {
			log.Info().Msg("r recovered")
			return nil
		}
	}

	return errors.New("R recovery failed")
}

// Stop stops the invoice tracker.
func (it *InvoiceTracker) Stop() {
	it.once.Do(func() {
		log.Debug().Msg("Stopping...")
		close(it.stop)
	})
}
