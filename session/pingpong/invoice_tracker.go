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
	"encoding/json"
	stdErr "errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
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

// ErrInvoiceSendMaxFailCountReached indicates that we did not sent an exchange message in time.
var ErrInvoiceSendMaxFailCountReached = errors.New("did not sent a new exchange message")

// ErrFirstInvoiceSendTimeout indicates that first invoice was not sent.
var ErrFirstInvoiceSendTimeout = errors.New("did not sent first invoice")

// ErrExchangeValidationFailed indicates that there was an error with the exchange signature.
var ErrExchangeValidationFailed = errors.New("exchange validation failed")

// ErrConsumerNotRegistered represents the error that the consumer is not registered
var ErrConsumerNotRegistered = errors.New("consumer not registered")

const providerFirstInvoiceTolerance = 0.8

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
	StoreR(providerIdentity identity.Identity, agreementID uint64, r string) error
	GetR(providerID identity.Identity, agreementID uint64) (string, error)
}

type accountantPromiseStorage interface {
	Store(providerID identity.Identity, accountantID common.Address, promise AccountantPromise) error
	Get(providerID identity.Identity, accountantID common.Address) (AccountantPromise, error)
}

type accountantCaller interface {
	RequestPromise(rp RequestPromise) (crypto.Promise, error)
	RevealR(r string, provider string, agreementID uint64) error
}

type settler func(providerID identity.Identity, accountantID common.Address) error

type sentInvoice struct {
	invoice crypto.Invoice
	r       []byte
}

// DataTransferred represents the data transfered in a session.
type DataTransferred struct {
	Up, Down uint64
}

func (dt DataTransferred) sum() uint64 {
	return dt.Up + dt.Down
}

// InvoiceTracker keeps tab of invoices and sends them to the consumer.
type InvoiceTracker struct {
	stop                       chan struct{}
	accountantFailureCount     uint64
	accountantFailureCountLock sync.Mutex

	notReceivedExchangeMessageCount uint64
	notSentExchangeMessageCount     uint64
	exchangeMessageCountLock        sync.Mutex

	maxNotReceivedExchangeMessages uint64
	maxNotSentExchangeMessages     uint64
	once                           sync.Once
	rnd                            *rand.Rand
	agreementID                    uint64
	lastExchangeMessage            crypto.ExchangeMessage
	transactorFee                  registry.FeesResponse
	invoicesSent                   map[string]sentInvoice
	invoiceLock                    sync.Mutex
	deps                           InvoiceTrackerDeps

	dataTransferred     DataTransferred
	dataTransferredLock sync.Mutex
}

type encryption interface {
	Decrypt(addr common.Address, encrypted []byte) ([]byte, error)
	Encrypt(addr common.Address, plaintext []byte) ([]byte, error)
}

// InvoiceTrackerDeps contains all the deps needed for invoice tracker.
type InvoiceTrackerDeps struct {
	Proposal                   market.ServiceProposal
	Peer                       identity.Identity
	PeerInvoiceSender          PeerInvoiceSender
	InvoiceStorage             providerInvoiceStorage
	TimeTracker                timeTracker
	ChargePeriodLeeway         time.Duration
	ChargePeriod               time.Duration
	ExchangeMessageChan        chan crypto.ExchangeMessage
	ExchangeMessageWaitTimeout time.Duration
	FirstInvoiceSendDuration   time.Duration
	FirstInvoiceSendTimeout    time.Duration
	ProviderID                 identity.Identity
	AccountantID               common.Address
	AccountantCaller           accountantCaller
	AccountantPromiseStorage   accountantPromiseStorage
	Registry                   string
	MaxAccountantFailureCount  uint64
	MaxAllowedAccountantFee    uint16
	BlockchainHelper           bcHelper
	EventBus                   eventbus.EventBus
	FeeProvider                feeProvider
	ChannelAddressCalculator   channelAddressCalculator
	Settler                    settler
	SessionID                  string
	Encryption                 encryption
}

// NewInvoiceTracker creates a new instance of invoice tracker.
func NewInvoiceTracker(
	itd InvoiceTrackerDeps) *InvoiceTracker {
	return &InvoiceTracker{
		stop:                           make(chan struct{}),
		deps:                           itd,
		maxNotReceivedExchangeMessages: calculateMaxNotReceivedExchangeMessageCount(itd.ChargePeriodLeeway, itd.ChargePeriod),
		maxNotSentExchangeMessages:     calculateMaxNotSentExchangeMessageCount(itd.ChargePeriodLeeway, itd.ChargePeriod),
		invoicesSent:                   make(map[string]sentInvoice),
		rnd:                            rand.New(rand.NewSource(1)),
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
	it.rnd.Seed(time.Now().UnixNano())
	it.agreementID = it.rnd.Uint64()
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

	it.lastExchangeMessage = em
	it.markInvoicePaid(em.Promise.Hashlock)
	it.resetNotReceivedExchangeMessageCount()
	it.resetNotSentExchangeMessageCount()

	// incase of zero payment, we'll just skip going to the accountant
	if isServiceFree(it.deps.Proposal.PaymentMethod) {
		return nil
	}

	err = it.revealPromise()
	switch errors.Cause(err) {
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

	err = it.requestPromise(invoice.r, em)
	switch errors.Cause(err) {
	case errHandled:
		return nil
	default:
		return err
	}
}

var errHandled = errors.New("error handled, please skip")

func (it *InvoiceTracker) updateFee() {
	fees, err := it.deps.FeeProvider.FetchSettleFees()
	if err != nil {
		log.Warn().Err(err).Msg("could not fetch fees, ignoring")
		return
	}

	it.transactorFee = fees
}

func (it *InvoiceTracker) requestPromise(r []byte, em crypto.ExchangeMessage) error {
	if !it.transactorFee.IsValid() {
		it.updateFee()
	}

	details := rRecoveryDetails{
		R:           hex.EncodeToString(r),
		AgreementID: em.AgreementID,
	}

	bytes, err := json.Marshal(details)
	if err != nil {
		return errors.Wrap(err, "could not marshal R recovery details")
	}

	encrypted, err := it.deps.Encryption.Encrypt(it.deps.ProviderID.ToCommonAddress(), bytes)
	if err != nil {
		return errors.Wrap(err, "could not encrypt R")
	}

	request := RequestPromise{
		ExchangeMessage: em,
		TransactorFee:   it.transactorFee.Fee,
		RRecoveryData:   hex.EncodeToString(encrypted),
	}

	promise, err := it.deps.AccountantCaller.RequestPromise(request)
	handledErr := it.handleAccountantError(err)
	if handledErr != nil {
		return errors.Wrap(handledErr, "could not request promise")
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

	promise.R = r
	it.deps.EventBus.Publish(event.AppTopicAccountantPromise, event.AppEventAccountantPromise{
		Promise:      promise,
		AccountantID: it.deps.AccountantID,
		ProviderID:   it.deps.ProviderID,
	})
	it.deps.EventBus.Publish(sessionEvent.AppTopicSessionTokensEarned, sessionEvent.AppEventSessionTokensEarned{
		ProviderID: it.deps.ProviderID,
		SessionID:  it.deps.SessionID,
		Total:      em.AgreementTotal,
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
	handledErr := it.handleAccountantError(err)
	if handledErr != nil {
		return errors.Wrap(handledErr, "could not reveal R")
	}

	accountantPromise.Revealed = true
	err = it.deps.AccountantPromiseStorage.Store(it.deps.ProviderID, it.deps.AccountantID, accountantPromise)
	if err != nil {
		return errors.Wrap(err, "could not store accountant promise")
	}

	return nil
}

// Start stars the invoice tracker
func (it *InvoiceTracker) Start() error {
	log.Debug().Msg("Starting...")
	it.deps.TimeTracker.StartTracking()

	if err := it.deps.EventBus.SubscribeAsync(sessionEvent.AppTopicDataTransferred, it.consumeDataTransferredEvent); err != nil {
		return err
	}

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
	it.transactorFee = fees

	fee, err := it.deps.BlockchainHelper.GetAccountantFee(it.deps.AccountantID)
	if err != nil {
		return errors.Wrap(err, "could not get accountants fee")
	}

	if fee > it.deps.MaxAllowedAccountantFee {
		log.Error().Msgf("Accountant fee too large, asking for %v where %v is the limit", fee, it.deps.MaxAllowedAccountantFee)
		return ErrAccountantFeeTooLarge
	}

	it.generateAgreementID()

	emErrors := make(chan error)
	go func() {
		emErrors <- it.listenForExchangeMessages()
	}()

	// on session close, try and reveal the promise before exiting
	defer it.revealPromise()

	err = it.sendFirstInvoice()
	if err != nil {
		return fmt.Errorf("could not send first invoice: %w", err)
	}

	for {
		select {
		case <-it.stop:
			return nil
		case <-time.After(it.deps.ChargePeriod):
			err := it.sendInvoice()
			if err != nil {
				return fmt.Errorf("sending of invoice failed: %w", err)
			}
		case emErr := <-emErrors:
			if emErr != nil {
				return errors.Wrap(emErr, "failed to get exchange message")
			}
		}
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

func (it *InvoiceTracker) generateR() []byte {
	r := make([]byte, 32)
	crand.Read(r)
	return r
}

func (it *InvoiceTracker) sendInvoice() error {
	if it.getNotSentExchangeMessageCount() >= it.maxNotSentExchangeMessages {
		return ErrInvoiceSendMaxFailCountReached
	}

	if it.getNotReceivedExchangeMessageCount() >= it.maxNotReceivedExchangeMessages {
		return ErrExchangeWaitTimeout
	}

	shouldBe := CalculatePaymentAmount(it.deps.TimeTracker.Elapsed(), it.getDataTransferred(), it.deps.Proposal.PaymentMethod)

	// In case we're sending a first invoice, there might be a big missmatch percentage wise on the consumer side.
	// This is due to the fact that both payment providers start at different times.
	// To compensate for this, be a bit more lenient on the first invoice - ask for a reduced amount.
	// Over the long run, this becomes redundant as the difference should become miniscule.
	if it.lastExchangeMessage.AgreementTotal == 0 {
		shouldBe = uint64(math.Trunc(float64(shouldBe) * providerFirstInvoiceTolerance))
		log.Debug().Msgf("Being lenient for the first payment, asking for %v", shouldBe)
	}

	r := it.generateR()
	invoice := crypto.CreateInvoice(it.agreementID, shouldBe, 0, r)
	invoice.Provider = it.deps.ProviderID.Address
	err := it.deps.PeerInvoiceSender.Send(invoice)
	if err != nil {
		if stdErr.Is(err, p2p.ErrSendTimeout) {
			log.Warn().Err(err).Msg("Marking invoice as not sent")
			it.markExchangeMessageNotSent()
			return nil
		}
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

func (it *InvoiceTracker) sendFirstInvoice() error {
	timeout := time.After(it.deps.FirstInvoiceSendTimeout)
	for {
		select {
		case <-it.stop:
			return nil
		case <-timeout:
			return ErrFirstInvoiceSendTimeout
		case <-time.After(it.deps.FirstInvoiceSendDuration):
			err := it.sendInvoice()
			if stdErr.Is(err, p2p.ErrHandlerNotFound) {
				continue
			}
			return err
		}
	}
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

func (it *InvoiceTracker) recoverR(aerr accountantError) error {
	log.Info().Msg("Recovering R...")
	decoded, err := hex.DecodeString(aerr.Data())
	if err != nil {
		return errors.Wrap(err, "could not decode R recovery details")
	}

	decrypted, err := it.deps.Encryption.Decrypt(it.deps.ProviderID.ToCommonAddress(), decoded)
	if err != nil {
		return errors.Wrap(err, "could not decrypt R details")
	}

	res := rRecoveryDetails{}
	err = json.Unmarshal(decrypted, &res)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal R details")
	}

	log.Info().Msg("R recovered, will reveal...")
	err = it.deps.AccountantCaller.RevealR(res.R, it.deps.ProviderID.Address, res.AgreementID)
	if err != nil {
		return errors.Wrap(err, "could not reveal R")
	}

	log.Info().Msg("R recovered successfully")
	return nil
}

func (it *InvoiceTracker) handleAccountantError(err error) error {
	if err == nil {
		it.resetAccountantFailureCount()
		return nil
	}

	switch {
	case stdErr.Is(err, ErrNeedsRRecovery):
		var aer AccountantErrorResponse
		ok := stdErr.As(err, &aer)
		if !ok {
			return errors.New("could not cast errNeedsRecovery to accountantError")
		}
		return it.recoverR(aer)
	case stdErr.Is(err, ErrAccountantNoPreviousPromise):
		log.Info().Msg("no previous promise on accountant, will mark R as revealed")
		return nil
	case stdErr.Is(err, ErrAccountantHashlockMissmatch), stdErr.Is(err, ErrAccountantPreviousRNotRevealed):
		// These should basicly be obsolete with the introduction of R recovery. Will remove in the future.
		// For now though, handle as ignorable.
		fallthrough
	case
		stdErr.Is(err, ErrAccountantInternal),
		stdErr.Is(err, ErrAccountantNotFound),
		stdErr.Is(err, ErrAccountantMalformedJSON):
		// these are ignorable, we'll eventually fail
		if it.incrementAccountantFailureCount() > it.deps.MaxAccountantFailureCount {
			return err
		}
		log.Warn().Err(err).Msg("accountant error, will retry")
		return errHandled
	case stdErr.Is(err, ErrAccountantProviderBalanceExhausted):
		go func() {
			settleErr := it.deps.Settler(it.deps.ProviderID, it.deps.AccountantID)
			if settleErr != nil {
				log.Err(settleErr).Msgf("settling failed")
			}
		}()
		if it.incrementAccountantFailureCount() > it.deps.MaxAccountantFailureCount {
			return err
		}
		log.Warn().Err(err).Msg("out of balance, will try settling")
		return errHandled
	case
		stdErr.Is(err, ErrAccountantInvalidSignature),
		stdErr.Is(err, ErrAccountantPaymentValueTooLow),
		stdErr.Is(err, ErrAccountantPromiseValueTooLow),
		stdErr.Is(err, ErrAccountantOverspend):
		// these are critical, return and cancel session
		return err
	default:
		log.Err(err).Msgf("unknown accountant error encountered")
		return err
	}
}

func (it *InvoiceTracker) incrementAccountantFailureCount() uint64 {
	it.accountantFailureCountLock.Lock()
	defer it.accountantFailureCountLock.Unlock()
	it.accountantFailureCount++
	log.Trace().Msgf("accountant error count %v/%v", it.accountantFailureCount, it.deps.MaxAccountantFailureCount)
	return it.accountantFailureCount
}

func (it *InvoiceTracker) resetAccountantFailureCount() {
	it.accountantFailureCountLock.Lock()
	defer it.accountantFailureCountLock.Unlock()
	it.accountantFailureCount = 0
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

// Stop stops the invoice tracker.
func (it *InvoiceTracker) Stop() {
	it.once.Do(func() {
		log.Debug().Msg("Stopping...")
		_ = it.deps.EventBus.Unsubscribe(sessionEvent.AppTopicDataTransferred, it.consumeDataTransferredEvent)
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
