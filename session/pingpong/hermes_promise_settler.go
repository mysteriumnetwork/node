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
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mysteriumnetwork/node/config"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

type settlementHistoryStorage interface {
	Store(she SettlementHistoryEntry) error
}

type providerChannelStatusProvider interface {
	GetHermesFee(chainID int64, hermesAddress common.Address) (uint16, error)
	CalculateHermesFee(chainID int64, hermesAddress common.Address, value *big.Int) (*big.Int, error)
	GetMystBalance(chainID int64, mystAddress, identity common.Address) (*big.Int, error)
	GetProvidersWithdrawalChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
	FilterPromiseSettledEventByChannelID(chainID int64, from uint64, to *uint64, hermesID common.Address, providerAddresses [][32]byte) ([]bindings.HermesImplementationPromiseSettled, error)
	HeaderByNumber(chainID int64, number *big.Int) (*types.Header, error)
}

type paySettler interface {
	PayAndSettle(r []byte, em crypto.ExchangeMessage, providerID identity.Identity, sessionID string) <-chan error
}

type ks interface {
	Accounts() []accounts.Account
	SignHash(a accounts.Account, hash []byte) ([]byte, error)
}

type registrationStatusProvider interface {
	GetRegistrationStatus(chainID int64, id identity.Identity) (registry.RegistrationStatus, error)
}

type promiseStorage interface {
	Get(chainID int64, channelID string) (HermesPromise, error)
	Delete(promise HermesPromise) error
}

type transactor interface {
	SettleAndRebalance(hermesID, providerID string, promise crypto.Promise) (string, error)
	SettleWithBeneficiary(id, beneficiary, hermesID string, promise crypto.Promise) (string, error)
	PayAndSettle(hermesID, providerID string, promise crypto.Promise, beneficiary string, beneficiarySignature string) (string, error)
	SettleIntoStake(hermesID, providerID string, promise crypto.Promise) (string, error)
	FetchSettleFees(chainID int64) (registry.FeesResponse, error)
	GetQueueStatus(ID string) (registry.QueueResponse, error)
}

type hermesChannelProvider interface {
	Get(chainID int64, id identity.Identity, hermesID common.Address) (HermesChannel, bool)
	Fetch(chainID int64, id identity.Identity, hermesID common.Address) (HermesChannel, error)
}

type receivedPromise struct {
	provider    identity.Identity
	hermesID    common.Address
	promise     crypto.Promise
	beneficiary common.Address
}

// HermesPromiseSettler is responsible for settling the hermes promises.
type HermesPromiseSettler interface {
	ForceSettle(chainID int64, providerID identity.Identity, hermesID common.Address) error
	SettleWithBeneficiary(chainID int64, providerID identity.Identity, beneficiary, hermesID common.Address) error
	SettleIntoStake(chainID int64, providerID identity.Identity, hermesID common.Address) error
	GetHermesFee(chainID int64, hermesID common.Address) (uint16, error)
	Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error
}

// hermesPromiseSettler is responsible for settling the hermes promises.
type hermesPromiseSettler struct {
	bc                         providerChannelStatusProvider
	config                     HermesPromiseSettlerConfig
	lock                       sync.RWMutex
	registrationStatusProvider registrationStatusProvider
	ks                         ks
	transactor                 transactor
	channelProvider            hermesChannelProvider
	settlementHistoryStorage   settlementHistoryStorage
	hermesURLGetter            hermesURLGetter
	hermesCallerFactory        HermesCallerFactory
	addressProvider            addressProvider
	paySettler                 paySettler
	promiseStorage             promiseStorage
	publisher                  eventbus.Publisher
	// TODO: Consider adding chain ID to this as well.
	currentState map[identity.Identity]settlementState
	settleQueue  chan receivedPromise
	stop         chan struct{}
	once         sync.Once
}

// HermesPromiseSettlerConfig configures the hermes promise settler accordingly.
type HermesPromiseSettlerConfig struct {
	Threshold               float64
	L1ChainID               int64
	L2ChainID               int64
	SettlementCheckInterval time.Duration
	SettlementCheckTimeout  time.Duration
}

// NewHermesPromiseSettler creates a new instance of hermes promise settler.
func NewHermesPromiseSettler(transactor transactor, promiseStorage promiseStorage, paySettler paySettler, addressProvider addressProvider, hermesCallerFactory HermesCallerFactory, hermesURLGetter hermesURLGetter, channelProvider hermesChannelProvider, providerChannelStatusProvider providerChannelStatusProvider, registrationStatusProvider registrationStatusProvider, ks ks, settlementHistoryStorage settlementHistoryStorage, publisher eventbus.Publisher, config HermesPromiseSettlerConfig) *hermesPromiseSettler {
	return &hermesPromiseSettler{
		bc:                         providerChannelStatusProvider,
		ks:                         ks,
		registrationStatusProvider: registrationStatusProvider,
		config:                     config,
		currentState:               make(map[identity.Identity]settlementState),
		channelProvider:            channelProvider,
		settlementHistoryStorage:   settlementHistoryStorage,
		hermesCallerFactory:        hermesCallerFactory,
		hermesURLGetter:            hermesURLGetter,
		addressProvider:            addressProvider,
		promiseStorage:             promiseStorage,
		paySettler:                 paySettler,
		publisher:                  publisher,
		// defaulting to a queue of 5, in case we have a few active identities.
		settleQueue: make(chan receivedPromise, 5),
		stop:        make(chan struct{}),
		transactor:  transactor,
	}
}

// GetHermesFee fetches the hermes fee.
func (aps *hermesPromiseSettler) GetHermesFee(chainID int64, hermesID common.Address) (uint16, error) {
	return aps.bc.GetHermesFee(chainID, hermesID)
}

// loadInitialState loads the initial state for the given identity. Inteded to be called on service start.
func (aps *hermesPromiseSettler) loadInitialState(chainID int64, id identity.Identity) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	if _, ok := aps.currentState[id]; ok {
		log.Info().Msgf("State for %v already loaded, skipping", id)
		return nil
	}

	status, err := aps.registrationStatusProvider.GetRegistrationStatus(chainID, id)
	if err != nil {
		return fmt.Errorf("could not check registration status for %v: %w", id, err)
	}

	if status != registry.Registered {
		log.Info().Msgf("Provider %v not registered, skipping", id)
		return nil
	}

	aps.currentState[id] = settlementState{
		registered: true,
	}
	return nil
}

// Subscribe subscribes the hermes promise settler to the appropriate events
func (aps *hermesPromiseSettler) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(nodevent.AppTopicNode, aps.handleNodeEvent)
	if err != nil {
		return fmt.Errorf("could not subscribe to node status event: %w", err)
	}

	err = bus.SubscribeAsync(registry.AppTopicIdentityRegistration, aps.handleRegistrationEvent)
	if err != nil {
		return fmt.Errorf("could not subscribe to registration event: %w", err)
	}

	err = bus.SubscribeAsync(servicestate.AppTopicServiceStatus, aps.handleServiceEvent)
	if err != nil {
		return fmt.Errorf("could not subscribe to service status event: %w", err)
	}

	err = bus.SubscribeAsync(event.AppTopicSettlementRequest, aps.handleSettlementEvent)
	if err != nil {
		return fmt.Errorf("could not subscribe to settlement event: %w", err)
	}

	err = bus.SubscribeAsync(event.AppTopicHermesPromise, aps.handleHermesPromiseReceived)
	if err != nil {
		return fmt.Errorf("could not subscribe to hermes promise event: %w", err)
	}
	return nil
}

func (aps *hermesPromiseSettler) handleSettlementEvent(event event.AppEventSettlementRequest) {
	err := aps.ForceSettle(event.ChainID, event.ProviderID, event.HermesID)
	if err != nil {
		log.Error().Err(err).Msg("could not settle promise")
	}
}

func (aps *hermesPromiseSettler) chainID() int64 {
	return config.GetInt64(config.FlagChainID)
}

func (aps *hermesPromiseSettler) handleServiceEvent(event servicestate.AppEventServiceStatus) {
	switch event.Status {
	case string(servicestate.Running):
		err := aps.loadInitialState(aps.chainID(), identity.FromAddress(event.ProviderID))
		if err != nil {
			log.Error().Err(err).Msgf("could not load initial state for provider %v", event.ProviderID)
		}
	default:
		log.Debug().Msgf("Ignoring service event with status %v", event.Status)
	}
}

func (aps *hermesPromiseSettler) handleNodeEvent(payload nodevent.Payload) {
	if payload.Status == nodevent.StatusStarted {
		aps.handleNodeStart()
		return
	}

	if payload.Status == nodevent.StatusStopped {
		aps.handleNodeStop()
		return
	}
}

func (aps *hermesPromiseSettler) handleRegistrationEvent(payload registry.AppEventIdentityRegistration) {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	if payload.Status != registry.Registered {
		log.Debug().Msgf("Ignoring event %v for provider %q", payload.Status.String(), payload.ID)
		return
	}
	log.Info().Msgf("Identity registration event received for provider %q", payload.ID)

	s := aps.currentState[payload.ID]
	s.registered = true
	aps.currentState[payload.ID] = s
	log.Info().Msgf("Identity registration event handled for provider %q", payload.ID)
}

func (aps *hermesPromiseSettler) handleHermesPromiseReceived(apep event.AppEventHermesPromise) {
	id := apep.ProviderID
	log.Info().Msgf("Received hermes promise for %q", id)
	aps.lock.Lock()
	defer aps.lock.Unlock()

	s, ok := aps.currentState[apep.ProviderID]
	if !ok {
		log.Error().Msgf("Have no info on provider %q, skipping", id)
		return
	}
	if !s.registered {
		log.Error().Msgf("provider %q not registered, skipping", id)
		return
	}

	var channel HermesChannel
	hc, ok := aps.channelProvider.Get(apep.Promise.ChainID, id, apep.HermesID)
	if ok {
		channel = hc
	} else {
		hc, err := aps.channelProvider.Fetch(apep.Promise.ChainID, id, apep.HermesID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			log.Error().Err(err).Msgf("could not sync state for provider %v, hermesID %v", apep.ProviderID, apep.HermesID.Hex())
			return
		}
		channel = hc
	}

	log.Info().Msgf("Hermes %q promise state updated for provider %q", apep.HermesID.Hex(), id)

	if s.needsSettling(aps.config.Threshold, channel) {
		log.Info().Msgf("Starting auto settle for provider %v", id)
		aps.initiateSettling(channel)
	}
}

func (aps *hermesPromiseSettler) initiateSettling(channel HermesChannel) {
	hexR, err := hex.DecodeString(channel.lastPromise.R)
	if err != nil {
		log.Error().Err(fmt.Errorf("could encode R: %w", err))
		return
	}
	channel.lastPromise.Promise.R = hexR

	aps.settleQueue <- receivedPromise{
		hermesID:    channel.HermesID,
		provider:    channel.Identity,
		promise:     channel.lastPromise.Promise,
		beneficiary: channel.Beneficiary,
	}
}

func (aps *hermesPromiseSettler) listenForSettlementRequests() {
	log.Info().Msg("Listening for settlement events")
	defer log.Info().Msg("Stopped listening for settlement events")

	for {
		select {
		case <-aps.stop:
			return
		case p := <-aps.settleQueue:
			channel, found := aps.channelProvider.Get(p.promise.ChainID, p.provider, p.hermesID)
			if !found {
				continue
			}
			go aps.settle(
				func(promise crypto.Promise) (string, error) {
					return aps.transactor.SettleAndRebalance(p.hermesID.Hex(), p.provider.Address, promise)
				},
				p.provider,
				p.hermesID,
				p.promise,
				p.beneficiary,
				channel.Channel.Settled,
			)
		}
	}
}

// SettleIntoStake settles the promise but transfers the money to stake increase, not to beneficiary.
func (aps *hermesPromiseSettler) SettleIntoStake(chainID int64, providerID identity.Identity, hermesID common.Address) error {
	channel, found := aps.channelProvider.Get(chainID, providerID, hermesID)
	if !found {
		return ErrNothingToSettle
	}

	hexR, err := hex.DecodeString(channel.lastPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R: %w", err)
	}
	channel.lastPromise.Promise.R = hexR
	return aps.settle(
		func(promise crypto.Promise) (string, error) {
			return aps.transactor.SettleIntoStake(hermesID.Hex(), providerID.Address, promise)
		},
		providerID,
		hermesID,
		channel.lastPromise.Promise,
		channel.Beneficiary,
		channel.Channel.Settled,
	)
}

// ErrNothingToSettle indicates that there is nothing to settle.
var ErrNothingToSettle = errors.New("nothing to settle for the given provider")

// ForceSettle forces the settlement for a provider
func (aps *hermesPromiseSettler) ForceSettle(chainID int64, providerID identity.Identity, hermesID common.Address) error {
	channel, found := aps.channelProvider.Get(chainID, providerID, hermesID)
	if !found {
		return ErrNothingToSettle
	}

	hexR, err := hex.DecodeString(channel.lastPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R: %w", err)
	}

	channel.lastPromise.Promise.R = hexR
	return aps.settle(
		func(promise crypto.Promise) (string, error) {
			return aps.transactor.SettleAndRebalance(hermesID.Hex(), providerID.Address, promise)
		},
		providerID,
		hermesID,
		channel.lastPromise.Promise,
		channel.Beneficiary,
		channel.Channel.Settled,
	)
}

// ForceSettle forces the settlement for a provider
func (aps *hermesPromiseSettler) SettleWithBeneficiary(chainID int64, providerID identity.Identity, beneficiary, hermesID common.Address) error {
	channel, found := aps.channelProvider.Get(chainID, providerID, hermesID)
	if !found {
		return ErrNothingToSettle
	}

	hexR, err := hex.DecodeString(channel.lastPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R: %w", err)
	}

	channel.lastPromise.Promise.R = hexR
	return aps.settle(
		func(promise crypto.Promise) (string, error) {
			return aps.transactor.SettleWithBeneficiary(providerID.Address, beneficiary.Hex(), hermesID.Hex(), promise)
		},
		providerID,
		hermesID,
		channel.lastPromise.Promise,
		beneficiary,
		channel.Channel.Settled,
	)
}

// ErrSettleTimeout indicates that the settlement has timed out
var ErrSettleTimeout = errors.New("settle timeout")

func (aps *hermesPromiseSettler) updatePromiseWithLatestFee(hermesID common.Address, promise crypto.Promise) (crypto.Promise, error) {
	log.Debug().Msgf("Updating promise with latest fee. HermesID %v", hermesID.Hex())
	fees, err := aps.transactor.FetchSettleFees(promise.ChainID)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not fetch settle fees: %w", err)
	}

	hermesCaller, err := aps.getHermesCaller(promise.ChainID, hermesID)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not fetch settle fees: %w", err)
	}

	updatedPromise, err := hermesCaller.UpdatePromiseFee(promise, fees.Fee)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not update promise fee: %w", err)
	}
	updatedPromise.R = promise.R
	log.Debug().Msg("promise updated with latest fee")
	return updatedPromise, nil
}

func (aps *hermesPromiseSettler) Withdraw(
	fromChainID int64,
	toChainID int64,
	providerID identity.Identity,
	hermesID,
	beneficiary common.Address,
	amountToWithdraw *big.Int,
) error {
	if aps.isSettling(providerID) {
		return errors.New("provider already has settlement in progress")
	}

	aps.setSettling(providerID, true)
	log.Info().Msgf("Marked provider %v as requesting settlement", providerID)
	defer aps.setSettling(providerID, false)

	if toChainID == 0 {
		toChainID = aps.config.L1ChainID
	}

	if fromChainID != aps.config.L2ChainID {
		return fmt.Errorf("can only withdraw from chain with ID %v, requested with %v", aps.config.L2ChainID, fromChainID)
	}

	registry, err := aps.addressProvider.GetRegistryAddress(fromChainID)
	if err != nil {
		return err
	}
	channel, err := aps.addressProvider.GetChannelImplementation(fromChainID)
	if err != nil {
		return err
	}

	consumerChannelAddress, err := aps.addressProvider.GetArbitraryChannelAddress(hermesID, registry, channel, providerID)
	if err != nil {
		return fmt.Errorf("could not generate channel address: %w", err)
	}

	chid, err := crypto.GenerateProviderChannelIDForPayAndSettle(providerID.Address, hermesID.Hex())
	if err != nil {
		return fmt.Errorf("could not get channel id for pay and settle: %w", err)
	}

	// 0. check if previous withdrawal attempt exists
	promiseFromStorage, err := aps.promiseStorage.Get(toChainID, chid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info().Msg("no previous promise, will do a new attempt")
		} else {
			return err
		}
	} else {
		oldAmountToWithdraw, err := aps.calculateAmountToWithdrawFromPreviousPromise(providerID, promiseFromStorage)

		if oldAmountToWithdraw.Cmp(big.NewInt(0)) > 0 {
			err = aps.validateWithdrawalAmount(oldAmountToWithdraw)
			if err != nil {
				return err
			}

			return aps.payAndSettleTransactor(amountToWithdraw, beneficiary, providerID, chid, promiseFromStorage)
		}

		aps.deleteWithdrawnPromise(promiseFromStorage)
	}

	// 1. calculate amount to withdraw - check balance on consumer channel
	data, err := aps.getHermesData(fromChainID, hermesID, providerID.ToCommonAddress())
	if err != nil {
		return err
	}

	if amountToWithdraw == nil {
		amountToWithdraw = new(big.Int).Sub(data.Balance, new(big.Int).Sub(data.LatestPromise.Amount, data.Settled))
	}

	err = aps.validateWithdrawalAmount(amountToWithdraw)
	if err != nil {
		return err
	}

	// 2. issue a self promise
	msg, err := aps.issueSelfPromise(fromChainID, amountToWithdraw, data.LatestPromise.Amount, providerID, consumerChannelAddress, hermesID)
	if err != nil {
		return err
	}

	// 3. call hermes with the promise via the payandsettle endpoint
	ch := aps.paySettler.PayAndSettle(msg.Promise.R, *msg, providerID, "")
	err = <-ch
	if err != nil {
		return fmt.Errorf("could not call hermes pay and settle:%w", err)
	}

	// 4. fetch the promise from storage
	promiseFromStorage, err = aps.promiseStorage.Get(toChainID, chid)
	if err != nil {
		return err
	}

	aps.publisher.Publish(event.AppTopicWithdrawalRequested, event.AppEventWithdrawalRequested{
		ProviderID: providerID,
		HermesID:   hermesID,
		FromChain:  fromChainID,
		ToChain:    toChainID,
	})

	decodedR, err := hex.DecodeString(promiseFromStorage.R)
	if err != nil {
		return fmt.Errorf("could not decode R %w", err)
	}
	promiseFromStorage.Promise.R = decodedR

	return aps.payAndSettleTransactor(amountToWithdraw, beneficiary, providerID, chid, promiseFromStorage)
}

func (aps *hermesPromiseSettler) deleteWithdrawnPromise(promiseFromStorage HermesPromise) {
	err := aps.promiseStorage.Delete(promiseFromStorage)
	if err != nil {
		log.Err(err).Msg("could not delete withdrawal promise")
	}
}

func (aps *hermesPromiseSettler) payAndSettleTransactor(amountToWithdraw *big.Int, beneficiary common.Address, providerID identity.Identity, chid string, promiseFromStorage HermesPromise) error {
	decodedR, err := hex.DecodeString(promiseFromStorage.R)
	if err != nil {
		return fmt.Errorf("could not decode R %w", err)
	}
	promiseFromStorage.Promise.R = decodedR

	// 5. add the missing beneficiary signature
	payload := crypto.NewPayAndSettleBeneficiaryPayload(beneficiary, aps.config.L1ChainID, chid, promiseFromStorage.Promise.Amount, client.ToBytes32(promiseFromStorage.Promise.R))
	err = payload.Sign(aps.ks, providerID.ToCommonAddress())
	if err != nil {
		return fmt.Errorf("could not sign pay and settle payload: %w", err)
	}

	go func() {
		log.Info().Msg("caling mister transactor")
		err := aps.payAndSettle(
			func(promise crypto.Promise) (string, error) {
				return aps.transactor.PayAndSettle(promiseFromStorage.HermesID.Hex(), providerID.Address, promise, payload.Beneficiary.Hex(), hex.EncodeToString(payload.Signature))
			},
			providerID,
			promiseFromStorage.HermesID,
			promiseFromStorage.Promise,
			beneficiary,
			amountToWithdraw,
			promiseFromStorage,
		)
		if err != nil {
			log.Err(err).Msg("could not withdraw")
			return
		}
		log.Info().Msg("withdrawal complete")
	}()
	return nil
}

func (aps *hermesPromiseSettler) calculateAmountToWithdrawFromPreviousPromise(providerID identity.Identity, promiseFromStorage HermesPromise) (*big.Int, error) {
	ch, err := aps.bc.GetProvidersWithdrawalChannel(promiseFromStorage.Promise.ChainID, promiseFromStorage.HermesID, providerID.ToCommonAddress(), true)
	if err != nil {
		return nil, err
	}
	if ch.Settled == nil {
		ch.Settled = big.NewInt(0)
	}

	diff := big.NewInt(0).Sub(promiseFromStorage.Promise.Amount, ch.Settled)
	diff = diff.Abs(diff)
	return diff, nil
}

func (aps *hermesPromiseSettler) validateWithdrawalAmount(amount *big.Int) error {
	fees, err := aps.transactor.FetchSettleFees(aps.config.L1ChainID)
	if err != nil {
		return err
	}

	if fees.Fee.Cmp(amount) > 0 {
		return fmt.Errorf("transactors fee exceeds amount to withdraw. Fee %v, amount to withdraw %v", fees.Fee.String(), amount.String())
	}
	return nil
}

func (aps *hermesPromiseSettler) payAndSettle(
	settleFunc func(promise crypto.Promise) (string, error),
	provider identity.Identity,
	hermesID common.Address,
	promise crypto.Promise,
	beneficiary common.Address,
	withdrawalAmount *big.Int,
	promiseFromStorage HermesPromise,
) error {
	updatedPromise, err := aps.updatePromiseWithLatestFee(hermesID, promise)
	if err != nil {
		log.Error().Err(err).Msg("Could not update promise fee")
		return err
	}

	if updatedPromise.Fee.Cmp(withdrawalAmount) > 0 {
		log.Error().Fields(map[string]interface{}{
			"promiseAmount": updatedPromise.Amount.String(),
			"transactorFee": updatedPromise.Fee.String(),
		}).Err(err).Msg("Earned amount too small for withdrawal")
		return fmt.Errorf("amount too small for withdrawal. Need at least %v, have %v", updatedPromise.Fee.String(), withdrawalAmount.String())
	}

	id, err := settleFunc(updatedPromise)
	if err != nil {
		log.Error().Err(err).Msgf("Could not settle promise for %v", provider)
		return err
	}

	channelID, err := crypto.GenerateProviderChannelIDForPayAndSettle(provider.Address, hermesID.Hex())
	if err != nil {
		return fmt.Errorf("could not generate provider channel address: %w", err)
	}

	errCh := aps.listenForSettlement(hermesID, beneficiary, updatedPromise, provider, aps.toBytes32(channelID), id)
	return <-errCh
}

func (aps *hermesPromiseSettler) issueSelfPromise(chainID int64, amount, previousPromiseAmount *big.Int, providerID identity.Identity, consumerChannelAddress, hermesAddress common.Address) (*crypto.ExchangeMessage, error) {
	r := aps.generateR()
	agreementID := aps.generateAgreementID()
	invoice := crypto.CreateInvoice(agreementID, amount, big.NewInt(0), r, 1)
	invoice.Provider = providerID.ToCommonAddress().Hex()

	promise, err := crypto.CreatePromise(consumerChannelAddress.Hex(), chainID, big.NewInt(0).Add(amount, previousPromiseAmount), big.NewInt(0), invoice.Hashlock, aps.ks, providerID.ToCommonAddress())
	if err != nil {
		return nil, fmt.Errorf("could not create promise: %w", err)
	}

	promise.R = r

	msg, err := crypto.CreateExchangeMessageWithPromise(aps.config.L1ChainID, invoice, promise, hermesAddress.Hex(), aps.ks, providerID.ToCommonAddress())
	if err != nil {
		return nil, fmt.Errorf("could not get create exchange message: %w", err)
	}

	return msg, nil
}

func (aps *hermesPromiseSettler) generateR() []byte {
	r := make([]byte, 32)
	_, err := rand.Read(r)
	if err != nil {
		panic(err)
	}
	return r
}

func (aps *hermesPromiseSettler) generateAgreementID() *big.Int {
	agreementID := make([]byte, 32)
	_, err := rand.Read(agreementID)
	if err != nil {
		panic(err)
	}
	return new(big.Int).SetBytes(agreementID)
}

func (aps *hermesPromiseSettler) getHermesDataForProvider(chainID int64, hermesID, identity common.Address) (*ConsumerData, error) {
	caller, err := aps.getHermesCaller(chainID, hermesID)
	if err != nil {
		return nil, err
	}

	data, err := caller.GetProviderData(chainID, identity.Hex())
	if err != nil {
		return nil, err
	}

	return data.fillZerosIfBigIntNull(), nil
}

func (aps *hermesPromiseSettler) getHermesData(chainID int64, hermesID, identity common.Address) (*ConsumerData, error) {
	caller, err := aps.getHermesCaller(chainID, hermesID)
	if err != nil {
		return nil, err
	}

	data, err := caller.GetConsumerData(chainID, identity.Hex())
	if err != nil {
		return nil, err
	}

	if data.Balance.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("nothing to withdraw. Balance in channel %v is %v", data.ChannelID, data.Balance)
	}

	return data.fillZerosIfBigIntNull(), nil
}

func (aps *hermesPromiseSettler) settle(
	settleFunc func(promise crypto.Promise) (string, error),
	provider identity.Identity,
	hermesID common.Address,
	promise crypto.Promise,
	beneficiary common.Address,
	settled *big.Int,
) error {
	if aps.isSettling(provider) {
		return errors.New("provider already has settlement in progress")
	}

	aps.setSettling(provider, true)
	defer aps.setSettling(provider, false)
	log.Info().Msgf("Marked provider %v as requesting settlement", provider)

	updatedPromise, err := aps.updatePromiseWithLatestFee(hermesID, promise)
	if err != nil {
		aps.setSettling(provider, false)
		log.Error().Err(err).Msg("Could not update promise fee")
		return err
	}

	if settled == nil {
		settled = new(big.Int)
	}

	amountToSettle := new(big.Int).Sub(updatedPromise.Amount, settled)

	fee, err := aps.bc.CalculateHermesFee(promise.ChainID, hermesID, amountToSettle)
	if err != nil {
		aps.setSettling(provider, false)
		log.Error().Err(err).Msg("Could not calculate hermes fee")
		return err
	}

	totalFees := new(big.Int).Add(fee, updatedPromise.Fee)
	if totalFees.Cmp(amountToSettle) > 0 {
		aps.setSettling(provider, false)
		log.Error().Fields(map[string]interface{}{
			"amountToSettle": amountToSettle.String(),
			"promiseAmount":  updatedPromise.Amount.String(),
			"settled":        settled.String(),
			"transactorFee":  updatedPromise.Fee.String(),
			"hermesFee":      fee.String(),
			"totalFees":      totalFees.String(),
		}).Err(err).Msg("Earned amount too small for settling")
		return fmt.Errorf("settlement fees exceed earning amount. Please provide more service and try again. Current earnings: %v, current fees: %v", amountToSettle, totalFees)
	}

	id, err := settleFunc(updatedPromise)
	if err != nil {
		log.Error().Err(err).Msgf("Could not settle promise for %v", provider)
		return err
	}

	channelID, err := crypto.GenerateProviderChannelID(provider.Address, hermesID.Hex())
	if err != nil {
		return fmt.Errorf("could not generate provider channel address: %w", err)
	}

	errCh := aps.listenForSettlement(hermesID, beneficiary, updatedPromise, provider, aps.toBytes32(channelID), id)
	return <-errCh
}

func (aps *hermesPromiseSettler) listenForSettlement(hermesID, beneficiary common.Address, promise crypto.Promise, provider identity.Identity, providerChannelID [32]byte, queueID string) <-chan error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		t := time.After(aps.config.SettlementCheckTimeout)
		for {
			select {
			case <-aps.stop:
				return
			case <-t:
				return
			case <-time.After(aps.config.SettlementCheckInterval):
				// TODO: we could cache this internally and avoid further transactor calls.
				res, err := aps.transactor.GetQueueStatus(queueID)
				if err != nil {
					log.Err(err).Str("queueID", queueID).Msg("could not get queue status")
					break
				}

				state := strings.ToLower(res.State)
				// if queued, continue
				if state == "queue" {
					break
				}

				// if error, don't wait as it will never complete
				if state == "error" {
					errCh <- fmt.Errorf("transactor reported queue error for id %v", queueID)
					return
				}

				// at this point, state should be done. If it is something else, abort.
				if state != "done" {
					errCh <- fmt.Errorf("transactor reported unknown settlement state for id %v, state %v", queueID, state)
					return
				}

				info, err := aps.findSettlementInBCLogs(promise.ChainID, res.Hash, hermesID, providerChannelID)
				if err != nil {
					if errors.Is(err, errNoSettlementFound) {
						log.Warn().Fields(map[string]interface{}{
							"hermesID": hermesID.Hex(),
							"provider": provider.Address,
							"queueID":  queueID,
						}).Err(err).Msg("no settlement found, will try again later")
						break
					} else {
						errCh <- fmt.Errorf("could not get settlement event from bc: %w", err)
						return
					}
				}

				ch, err := aps.channelProvider.Fetch(promise.ChainID, provider, hermesID)
				if err != nil {
					log.Error().Err(err).Msgf("Resync failed for provider %v", provider)
				} else {
					log.Info().Msgf("Resync success for provider %v", provider)
				}

				she := SettlementHistoryEntry{
					TxHash:     common.HexToHash(res.Hash),
					ProviderID: provider,
					HermesID:   hermesID,
					// TODO: this should probably be either provider channel address or the consumer address from the promise, not truncated provider channel address.
					ChannelAddress: common.BytesToAddress(providerChannelID[:]),
					Time:           time.Now().UTC(),
					Promise:        promise,
					Beneficiary:    beneficiary,
					Amount:         info.AmountSentToBeneficiary,
					Fees:           info.Fees,
					TotalSettled:   ch.Channel.Settled,
				}

				err = aps.settlementHistoryStorage.Store(she)
				if err != nil {
					log.Error().Err(err).Msg("Could not store settlement history")
				}
				log.Info().Msgf("Settling complete for provider %v", provider)

				aps.publisher.Publish(event.AppTopicSettlementComplete, event.AppEventSettlementComplete{
					ProviderID: provider,
					HermesID:   hermesID,
					TxHash:     res.Hash,
					ChainID:    promise.ChainID,
				})
				return
			}
		}
	}()
	return errCh
}

func (aps *hermesPromiseSettler) toBytes32(providerAddress string) [32]byte {
	var arr [32]byte
	copy(arr[:], crypto.HexToBytes(providerAddress)[:32])
	return arr
}

var errNoSettlementFound = errors.New("no settlement found")

func (aps *hermesPromiseSettler) findSettlementInBCLogs(chainID int64, txHash string, hermesID common.Address, providerAddress [32]byte) (bindings.HermesImplementationPromiseSettled, error) {
	latest, err := aps.bc.HeaderByNumber(chainID, nil)
	if err != nil {
		return bindings.HermesImplementationPromiseSettled{}, err
	}
	blockNo := latest.Number.Uint64()
	from := aps.safeSub(blockNo, 800)

	filtered, err := aps.bc.FilterPromiseSettledEventByChannelID(chainID, from, nil, hermesID, [][32]byte{providerAddress})
	if err != nil {
		return bindings.HermesImplementationPromiseSettled{}, err
	}

	expected := common.BytesToHash(crypto.HexToBytes(txHash))
	for _, v := range filtered {
		log.Info().Str("expected", expected.Hex()).Str("got", v.Raw.TxHash.Hex()).Msg("filtering")
		if bytes.EqualFold(v.Raw.TxHash.Bytes(), expected.Bytes()) {
			return v, nil
		}
	}

	return bindings.HermesImplementationPromiseSettled{}, errNoSettlementFound
}

func (aps *hermesPromiseSettler) safeSub(a uint64, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}

func (aps *hermesPromiseSettler) isSettling(id identity.Identity) bool {
	aps.lock.RLock()
	defer aps.lock.RUnlock()
	v, ok := aps.currentState[id]
	if !ok {
		return false
	}

	return v.settleInProgress
}

func (aps *hermesPromiseSettler) setSettling(id identity.Identity, settling bool) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	v := aps.currentState[id]
	v.settleInProgress = settling
	aps.currentState[id] = v
}

func (aps *hermesPromiseSettler) handleNodeStart() {
	go aps.listenForSettlementRequests()

	for _, v := range aps.ks.Accounts() {
		addr := identity.FromAddress(v.Address.Hex())
		go func(address identity.Identity) {
			err := aps.loadInitialState(aps.chainID(), address)
			if err != nil {
				log.Error().Err(err).Msgf("could not load initial state for %v", addr)
			}
		}(addr)
	}
}

func (aps *hermesPromiseSettler) getHermesCaller(chainID int64, hermesID common.Address) (HermesHTTPRequester, error) {
	addr, err := aps.hermesURLGetter.GetHermesURL(chainID, hermesID)
	if err != nil {
		return nil, fmt.Errorf("could not get hermes URL: %w", err)
	}
	return aps.hermesCallerFactory(addr), nil
}

func (aps *hermesPromiseSettler) handleNodeStop() {
	aps.once.Do(func() {
		close(aps.stop)
	})
}

// settlementState earning calculations model
type settlementState struct {
	settleInProgress bool
	registered       bool
}

func (ss settlementState) needsSettling(threshold float64, channel HermesChannel) bool {
	if !ss.registered {
		return false
	}

	if ss.settleInProgress {
		return false
	}

	if channel.Channel.Stake.Cmp(big.NewInt(0)) == 0 {
		// if starting with zero stake, only settle one myst or more.
		return channel.UnsettledBalance().Cmp(big.NewInt(0).SetUint64(crypto.Myst)) == 1
	}

	floated := new(big.Float).SetInt(channel.availableBalance())
	calculatedThreshold := new(big.Float).Mul(big.NewFloat(threshold), floated)
	possibleEarnings := channel.UnsettledBalance()
	i, _ := calculatedThreshold.Int(nil)
	if possibleEarnings.Cmp(i) == -1 {
		return false
	}

	if channel.balance().Cmp(i) <= 0 {
		return true
	}

	return false
}
