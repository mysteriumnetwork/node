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
	crand "crypto/rand"
	"encoding/hex"
	stdErr "errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
)

// ErrConsumerPromiseValidationFailed represents an error where consumer tries to cheat us with incorrect promises.
var ErrConsumerPromiseValidationFailed = errors.New("consumer failed to issue promise for the correct amount")

// ErrHermesFeeTooLarge indicates that we do not allow hermess with such high fees
var ErrHermesFeeTooLarge = errors.New("hermes fee exceeds predefined limits")

// ErrHermesInactive indicates that the chosen hermes is not active
var ErrHermesInactive = errors.New("hermes is not active")

// ErrInvoiceExpired shows that the given invoice has already expired
var ErrInvoiceExpired = errors.New("invoice expired")

// ErrExchangeWaitTimeout indicates that we did not get an exchange message in time.
var ErrExchangeWaitTimeout = errors.New("did not get a new exchange message")

// ErrInvoiceSendMaxFailCountReached indicates that we did not sent an exchange message in time.
var ErrInvoiceSendMaxFailCountReached = errors.New("did not sent a new exchange message")

// ErrExchangeValidationFailed indicates that there was an error with the exchange signature.
var ErrExchangeValidationFailed = errors.New("exchange validation failed")

// ErrConsumerNotRegistered represents the error that the consumer is not registered
var ErrConsumerNotRegistered = errors.New("consumer not registered")

var providerFirstInvoiceValue = big.NewInt(1)

// PeerInvoiceSender allows to send invoices.
type PeerInvoiceSender interface {
	Send(crypto.Invoice) error
}

type hermesStatusChecker interface {
	GetHermesStatus(chainID int64, registryAddress common.Address, hermesID common.Address) (HermesStatus, error)
}

type providerInvoiceStorage interface {
	Get(providerIdentity, consumerIdentity identity.Identity) (crypto.Invoice, error)
	Store(providerIdentity, consumerIdentity identity.Identity, invoice crypto.Invoice) error
	StoreR(providerIdentity identity.Identity, agreementID *big.Int, r string) error
	GetR(providerID identity.Identity, agreementID *big.Int) (string, error)
}

type promiseHandler interface {
	RequestPromise(r []byte, em crypto.ExchangeMessage, providerID identity.Identity, sessionID string) <-chan error
}

type sentInvoice struct {
	invoice    crypto.Invoice
	r          []byte
	isCritical bool
}

// DataTransferred represents the data transferred in a session.
type DataTransferred struct {
	Up, Down uint64
}

func (dt DataTransferred) sum() uint64 {
	return dt.Up + dt.Down
}

// InvoiceTracker keeps tab of invoices and sends them to the consumer.
type InvoiceTracker struct {
	stop                   chan struct{}
	promiseErrors          chan error
	invoiceChannel         chan bool
	hermesFailureCount     uint64
	hermesFailureCountLock sync.Mutex

	notReceivedExchangeMessageCount uint64
	notSentExchangeMessageCount     uint64
	exchangeMessageCountLock        sync.Mutex

	maxNotReceivedExchangeMessages uint64
	maxNotSentExchangeMessages     uint64
	once                           sync.Once
	agreementID                    *big.Int
	firstInvoicePaid               bool
	invoicesSent                   map[string]sentInvoice
	invoiceLock                    sync.Mutex
	deps                           InvoiceTrackerDeps

	dataTransferred     DataTransferred
	dataTransferredLock sync.Mutex

	criticalInvoiceErrors chan error
	lastInvoiceSent       time.Duration
	invoiceDebounceRate   time.Duration

	lastExchangeMessage     crypto.ExchangeMessage
	lastExchangeMessageLock sync.Mutex
}

// InvoiceTrackerDeps contains all the deps needed for invoice tracker.
type InvoiceTrackerDeps struct {
	AgreedPrice                market.Price
	Peer                       identity.Identity
	PeerInvoiceSender          PeerInvoiceSender
	InvoiceStorage             providerInvoiceStorage
	TimeTracker                timeTracker
	ChargePeriodLeeway         time.Duration
	ExchangeMessageChan        chan crypto.ExchangeMessage
	ExchangeMessageWaitTimeout time.Duration
	ProviderID                 identity.Identity
	ConsumersHermesID          common.Address
	AddressProvider            addressProvider
	MaxHermesFailureCount      uint64
	MaxAllowedHermesFee        uint16
	HermesStatusChecker        hermesStatusChecker
	EventBus                   eventbus.EventBus
	SessionID                  string
	PromiseHandler             promiseHandler
	ChainID                    int64
	ChargePeriod               time.Duration
	LimitChargePeriod          time.Duration
	LimitNotPaidInvoice        *big.Int
	MaxNotPaidInvoice          *big.Int
	Observer                   observerApi
}

// NewInvoiceTracker creates a new instance of invoice tracker.
func NewInvoiceTracker(
	itd InvoiceTrackerDeps,
) *InvoiceTracker {
	return &InvoiceTracker{
		lastExchangeMessage: crypto.ExchangeMessage{
			Promise: crypto.Promise{
				Amount: new(big.Int),
				Fee:    new(big.Int),
			},
			AgreementID:    new(big.Int),
			AgreementTotal: new(big.Int),
		},
		stop:                           make(chan struct{}),
		deps:                           itd,
		maxNotReceivedExchangeMessages: calculateMaxNotReceivedExchangeMessageCount(itd.ChargePeriodLeeway, itd.ChargePeriod),
		maxNotSentExchangeMessages:     calculateMaxNotSentExchangeMessageCount(itd.ChargePeriodLeeway, itd.ChargePeriod),
		invoicesSent:                   make(map[string]sentInvoice),
		promiseErrors:                  make(chan error),
		criticalInvoiceErrors:          make(chan error),
		invoiceChannel:                 make(chan bool),
		invoiceDebounceRate:            time.Second * 5,
	}
}

func calculateMaxNotReceivedExchangeMessageCount(chargeLeeway, chargePeriod time.Duration) uint64 {
	return uint64(math.Round(float64(chargeLeeway) / float64(chargePeriod)))
}

func calculateMaxNotSentExchangeMessageCount(chargeLeeway, chargePeriod time.Duration) uint64 {
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

	if !it.firstInvoicePaid {
		it.firstInvoicePaid = true
	}

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

func (it *InvoiceTracker) generateAgreementID() {
	agreementID := make([]byte, 32)
	_, err := crand.Read(agreementID)
	if err != nil {
		panic(err)
	}
	it.agreementID = new(big.Int).SetBytes(agreementID)
}

func (it *InvoiceTracker) handleExchangeMessage(em crypto.ExchangeMessage) error {
	invoice, ok := it.getMarkedInvoice(em.Promise.Hashlock)
	if !ok {
		log.Debug().Msgf("consumer sent exchange message with missing expired hashlock %s, skipping", invoice.invoice.Hashlock)
		return ErrInvoiceExpired
	}

	err := it.validateExchangeMessage(em)
	if err != nil {
		return err
	}

	it.saveLastExchangeMessage(em)
	it.markInvoicePaid(em.Promise.Hashlock)
	it.resetNotReceivedExchangeMessageCount()
	it.resetNotSentExchangeMessageCount()

	// incase of zero payment, we'll just skip going to the hermes
	if it.deps.AgreedPrice.IsFree() {
		return nil
	}

	err = it.deps.InvoiceStorage.StoreR(it.deps.ProviderID, it.agreementID, hex.EncodeToString(invoice.r))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not store r: %s", hex.EncodeToString(invoice.r)))
	}
	errChan := it.deps.PromiseHandler.RequestPromise(invoice.r, em, it.deps.ProviderID, it.deps.SessionID)
	go it.handlePromiseErrors(errChan)
	return nil
}

// Start stars the invoice tracker
func (it *InvoiceTracker) Start() error {
	log.Debug().Msgf("Starting invoice tracker for session %s", it.deps.SessionID)
	it.deps.TimeTracker.StartTracking()

	if err := it.deps.EventBus.SubscribeWithUID(sessionEvent.AppTopicDataTransferred, it.deps.SessionID, it.consumeDataTransferredEvent); err != nil {
		return err
	}

	registry, err := it.deps.AddressProvider.GetRegistryAddress(it.deps.ChainID)
	if err != nil {
		return err
	}

	status, err := it.deps.HermesStatusChecker.GetHermesStatus(it.deps.ChainID, registry, it.deps.ConsumersHermesID)
	if err != nil {
		return fmt.Errorf("could not check hermes status: %w", err)
	}

	if !status.IsActive {
		log.Error().Msgf("Hermes(%v) is inactive", it.deps.ConsumersHermesID.Hex())
		return ErrHermesInactive
	}

	if status.Fee > it.deps.MaxAllowedHermesFee {
		log.Error().Msgf("Hermes fee too large, asking for %v where %v is the limit", status.Fee, it.deps.MaxAllowedHermesFee)
		return ErrHermesFeeTooLarge
	}

	it.generateAgreementID()

	emErrors := make(chan error)
	go func() {
		emErrors <- it.listenForExchangeMessages()
	}()

	err = it.sendInvoice(true)
	if err != nil {
		return fmt.Errorf("could not send first invoice: %w", err)
	}

	go it.sendInvoicesWhenNeeded(time.Second)
	for {
		select {
		case <-it.stop:
			return nil
		case critical := <-it.invoiceChannel:
			err := it.sendInvoice(critical)
			if err != nil {
				if stdErr.Is(err, p2p.ErrSendTimeout) {
					log.Warn().Err(err).Msg("Marking invoice as not sent")
					it.markExchangeMessageNotSent()
				} else {
					return fmt.Errorf("sending of invoice failed: %w", err)
				}
			}
		case err := <-it.criticalInvoiceErrors:
			return err
		case emErr := <-emErrors:
			if emErr != nil {
				return errors.Wrap(emErr, "failed to get exchange message")
			}
		case pErr := <-it.promiseErrors:
			err := it.handleHermesError(pErr)
			if err != nil {
				return fmt.Errorf("could not request promise: %w", err)
			}
		}
	}
}

func (it *InvoiceTracker) sendInvoicesWhenNeeded(interval time.Duration) {
	it.lastInvoiceSent = it.deps.TimeTracker.Elapsed()
	for {
		select {
		case <-it.stop:
			return
		case <-time.After(interval):
			currentlyElapsed := it.deps.TimeTracker.Elapsed()
			shouldBe := CalculatePaymentAmount(currentlyElapsed, it.getDataTransferred(), it.deps.AgreedPrice)
			lastEM := it.getLastExchangeMessage()
			diff := safeSub(shouldBe, lastEM.AgreementTotal)
			if diff.Cmp(it.deps.MaxNotPaidInvoice) >= 0 && currentlyElapsed-it.lastInvoiceSent > it.invoiceDebounceRate {
				it.lastInvoiceSent = it.deps.TimeTracker.Elapsed()
				it.invoiceChannel <- true

				it.updateMaxUnpaid()
			} else if currentlyElapsed-it.lastInvoiceSent > it.deps.ChargePeriod {
				it.lastInvoiceSent = it.deps.TimeTracker.Elapsed()
				it.invoiceChannel <- false

				it.updateTimer()
			}
		}
	}
}

const sessionInvoiceIncreaseSlope = 3

func (it *InvoiceTracker) updateMaxUnpaid() {
	limit := it.deps.LimitNotPaidInvoice
	if limit == nil || it.deps.MaxNotPaidInvoice.Cmp(limit) >= 0 {
		return
	}

	add := new(big.Int).Div(it.deps.MaxNotPaidInvoice, new(big.Int).SetInt64(sessionInvoiceIncreaseSlope))
	bigger := new(big.Int).Add(it.deps.MaxNotPaidInvoice, add)
	if bigger.Cmp(limit) > 0 {
		bigger = limit
	}

	it.deps.MaxNotPaidInvoice = bigger
	log.Debug().Str("invoice_amount", it.deps.MaxNotPaidInvoice.String()).Msg("Max invoice amount increased")
}

func (it *InvoiceTracker) updateTimer() {
	maxTime := it.deps.LimitChargePeriod
	if it.deps.ChargePeriod >= maxTime {
		return
	}

	newMaxTime := it.deps.ChargePeriod/sessionInvoiceIncreaseSlope + it.deps.ChargePeriod
	if newMaxTime > maxTime {
		newMaxTime = maxTime
	}
	it.deps.ChargePeriod = newMaxTime
	log.Debug().Int64("change_period (ms)", it.deps.ChargePeriod.Milliseconds()).Msg("Max charge period increased")
}

// WaitFirstInvoice waits for a first invoice to be paid.
func (it *InvoiceTracker) WaitFirstInvoice(wait time.Duration) error {
	timeout := time.After(wait)

	for {
		select {
		case <-time.After(10 * time.Millisecond):
			it.invoiceLock.Lock()
			paid := it.firstInvoicePaid
			it.invoiceLock.Unlock()
			if paid {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("failed waiting for first invoice")
		case <-it.stop:
			return nil
		}
	}
}

func (it *InvoiceTracker) handlePromiseErrors(ch <-chan error) {
	for err := range ch {
		it.promiseErrors <- err
	}
}

func (it *InvoiceTracker) markExchangeMessageNotReceived() {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	it.notReceivedExchangeMessageCount++
}

func (it *InvoiceTracker) markExchangeMessageNotSent() {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	it.notSentExchangeMessageCount++
}

func (it *InvoiceTracker) resetNotReceivedExchangeMessageCount() {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	it.notReceivedExchangeMessageCount = 0
}

func (it *InvoiceTracker) resetNotSentExchangeMessageCount() {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	it.notSentExchangeMessageCount = 0
}

func (it *InvoiceTracker) getNotReceivedExchangeMessageCount() uint64 {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	return it.notReceivedExchangeMessageCount
}

func (it *InvoiceTracker) getNotSentExchangeMessageCount() uint64 {
	it.exchangeMessageCountLock.Lock()
	defer it.exchangeMessageCountLock.Unlock()
	return it.notSentExchangeMessageCount
}

func (it *InvoiceTracker) saveLastExchangeMessage(em crypto.ExchangeMessage) {
	it.lastExchangeMessageLock.Lock()
	defer it.lastExchangeMessageLock.Unlock()
	it.lastExchangeMessage = em
}

func (it *InvoiceTracker) getLastExchangeMessage() crypto.ExchangeMessage {
	it.lastExchangeMessageLock.Lock()
	defer it.lastExchangeMessageLock.Unlock()
	return it.lastExchangeMessage
}

func (it *InvoiceTracker) chainID() int64 {
	return config.GetInt64(config.FlagChainID)
}

func (it *InvoiceTracker) sendInvoice(isCritical bool) error {
	if it.getNotSentExchangeMessageCount() >= it.maxNotSentExchangeMessages {
		return ErrInvoiceSendMaxFailCountReached
	}

	if it.getNotReceivedExchangeMessageCount() >= it.maxNotReceivedExchangeMessages {
		return ErrExchangeWaitTimeout
	}

	shouldBe := CalculatePaymentAmount(it.deps.TimeTracker.Elapsed(), it.getDataTransferred(), it.deps.AgreedPrice)

	lastEm := it.getLastExchangeMessage()
	if lastEm.AgreementTotal.Cmp(big.NewInt(0)) == 0 && shouldBe.Cmp(big.NewInt(0)) == 1 {
		// The first invoice should have minimal static value.
		shouldBe = providerFirstInvoiceValue
		log.Debug().Msgf("Being lenient for the first payment, asking for %v", shouldBe)
	}

	r, err := crypto.GenerateR()
	if err != nil {
		return fmt.Errorf("failed to generate R: %w", err)
	}
	invoice, err := crypto.CreateInvoice(it.agreementID, shouldBe, new(big.Int), r, it.chainID())
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	invoice.Provider = it.deps.ProviderID.Address
	err = it.deps.PeerInvoiceSender.Send(invoice)
	if err != nil {
		return err
	}

	it.markInvoiceSent(sentInvoice{
		invoice:    invoice,
		r:          r,
		isCritical: isCritical,
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
		if !ok {
			return
		}

		if inv.isCritical {
			log.Info().Msgf("did not get paid for invoice with hashlock %v, invoice is critical. Aborting.", inv.invoice.Hashlock)
			it.criticalInvoiceErrors <- fmt.Errorf("did not get paid for critical invoice with hashlock %v", inv.invoice.Hashlock)
			return
		}

		log.Info().Msgf("did not get paid for invoice with hashlock %v, incrementing failure count", inv.invoice.Hashlock)
		it.markInvoicePaid(hlock)
		it.markExchangeMessageNotReceived()
	case <-it.stop:
		return
	}
}

func (it *InvoiceTracker) handleHermesError(err error) error {
	if err == nil {
		it.resetHermesFailureCount()
		return nil
	}

	switch {
	case
		stdErr.Is(err, ErrHermesHashlockMissmatch),
		stdErr.Is(err, ErrHermesPreviousRNotRevealed),
		stdErr.Is(err, ErrHermesInternal),
		stdErr.Is(err, ErrHermesNotFound),
		stdErr.Is(err, ErrHermesMalformedJSON),
		stdErr.Is(err, ErrTooManyRequests):
		// these are ignorable, we'll eventually fail
		if it.incrementHermesFailureCount() > it.deps.MaxHermesFailureCount {
			return err
		}
		log.Warn().Err(err).Msg("hermes error, will retry")
		return nil
	case
		stdErr.Is(err, ErrHermesInvalidSignature),
		stdErr.Is(err, ErrHermesPaymentValueTooLow),
		stdErr.Is(err, ErrHermesPromiseValueTooLow),
		stdErr.Is(err, ErrHermesOverspend),
		stdErr.Is(err, ErrConsumerUnregistered):
		// these are critical, return and cancel session
		return err
	// under normal use, this should not occur. If it does, we should drop sessions until we settle because we're not getting paid.
	case stdErr.Is(err, ErrHermesProviderBalanceExhausted):
		hermes, err := it.deps.AddressProvider.GetActiveHermes(it.chainID())
		if err != nil {
			return err
		}
		it.deps.EventBus.Publish(
			event.AppTopicSettlementRequest,
			event.AppEventSettlementRequest{
				ChainID:    it.chainID(),
				HermesID:   hermes,
				ProviderID: it.deps.ProviderID,
			},
		)
		return err
	default:
		if it.incrementHermesFailureCount() > it.deps.MaxHermesFailureCount {
			return err
		}
		log.Warn().Err(err).Msg("unknown hermes error encountered, will retry")
		return nil
	}
}

func (it *InvoiceTracker) incrementHermesFailureCount() uint64 {
	it.hermesFailureCountLock.Lock()
	defer it.hermesFailureCountLock.Unlock()
	it.hermesFailureCount++
	log.Trace().Msgf("hermes error count %v/%v", it.hermesFailureCount, it.deps.MaxHermesFailureCount)
	return it.hermesFailureCount
}

func (it *InvoiceTracker) resetHermesFailureCount() {
	it.hermesFailureCountLock.Lock()
	defer it.hermesFailureCountLock.Unlock()
	it.hermesFailureCount = 0
}

func (it *InvoiceTracker) validateExchangeMessage(em crypto.ExchangeMessage) error {
	peerAddr := common.HexToAddress(it.deps.Peer.Address)
	if res := em.IsMessageValid(peerAddr); !res {
		return ErrExchangeValidationFailed
	}

	if em.ChainID != it.chainID() {
		return fmt.Errorf("invalid chain id in exchange message: expected %v, got %v", it.chainID(), em.ChainID)
	}

	signer, err := em.Promise.RecoverSigner()
	if err != nil {
		return errors.Wrap(err, "could not recover promise signature")
	}

	if signer.Hex() != peerAddr.Hex() {
		return errors.New("identity missmatch")
	}

	lastEm := it.getLastExchangeMessage()
	if em.Promise.Amount.Cmp(lastEm.Promise.Amount) == -1 {
		log.Warn().Msgf("Consumer sent an invalid amount. Expected < %v, got %v", lastEm.Promise.Amount, em.Promise.Amount)
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid amount")
	}

	registry, err := it.deps.AddressProvider.GetRegistryAddress(em.ChainID)
	if err != nil {
		return errors.Wrap(err, "could not get registry address")
	}

	hermesId := common.HexToAddress(em.HermesID)
	chimp, err := it.deps.AddressProvider.GetChannelImplementationForHermes(em.ChainID, hermesId)
	if err != nil {
		log.Err(err).Msgf("Failed to get channel implementation for hermes %s, using fallback", em.HermesID)
		hermesData, err := it.deps.Observer.GetHermesData(em.ChainID, hermesId)
		if err != nil {
			return errors.Wrap(err, "could not get channel implementation")
		}
		chimp = hermesData.ChannelImpl
	}

	addr, err := it.deps.AddressProvider.GetArbitraryChannelAddress(common.HexToAddress(em.HermesID), registry, chimp, it.deps.Peer.ToCommonAddress())
	if err != nil {
		return errors.Wrap(err, "could not generate channel address")
	}

	expectedChannel, err := hex.DecodeString(strings.TrimPrefix(addr.Hex(), "0x"))
	if err != nil {
		return errors.Wrap(err, "could not decode expected chanel")
	}

	if !bytes.Equal(expectedChannel, em.Promise.ChannelID) {
		log.Warn().Msgf("Consumer sent an invalid channel address. Expected %q, got %q", addr.Hex(), hex.EncodeToString(em.Promise.ChannelID))
		return errors.Wrap(ErrConsumerPromiseValidationFailed, "invalid channel address")
	}
	return nil
}

// Stop stops the invoice tracker.
func (it *InvoiceTracker) Stop() {
	it.once.Do(func() {
		log.Debug().Msgf("Stopping invoice tracker for session %s", it.deps.SessionID)
		_ = it.deps.EventBus.UnsubscribeWithUID(sessionEvent.AppTopicDataTransferred, it.deps.SessionID, it.consumeDataTransferredEvent)
		close(it.stop)
	})
}

func (it *InvoiceTracker) consumeDataTransferredEvent(e sessionEvent.AppEventDataTransferred) {
	// skip irrelevant sessions
	if !strings.EqualFold(e.ID, it.deps.SessionID) {
		return
	}

	// From a server perspective, bytes up are the actual bytes the client downloaded(aka the bytes we pushed to the consumer)
	// To lessen the confusion, I suggest having the bytes reversed on the session instance.
	// This way, the session will show that it downloaded the bytes in a manner that is easier to comprehend.
	it.updateDataTransfer(e.Down, e.Up)
}

func (it *InvoiceTracker) updateDataTransfer(up, down uint64) {
	it.dataTransferredLock.Lock()
	defer it.dataTransferredLock.Unlock()

	newUp := it.dataTransferred.Up
	if up > it.dataTransferred.Up {
		newUp = up
	}

	newDown := it.dataTransferred.Down
	if down > it.dataTransferred.Down {
		newDown = down
	}

	it.dataTransferred = DataTransferred{
		Up:   newUp,
		Down: newDown,
	}
}

func (it *InvoiceTracker) getDataTransferred() DataTransferred {
	it.dataTransferredLock.Lock()
	defer it.dataTransferredLock.Unlock()

	return it.dataTransferred
}
