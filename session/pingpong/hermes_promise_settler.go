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
	"github.com/mysteriumnetwork/go-rest/apierror"
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
	"github.com/mysteriumnetwork/payments/observer"
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
	TransactionReceipt(chainID int64, hash common.Hash) (*types.Receipt, error)
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
	maxFee      *big.Int
}

// HermesPromiseSettler is responsible for settling the hermes promises.
type HermesPromiseSettler interface {
	ForceSettle(chainID int64, providerID identity.Identity, hermesID ...common.Address) error
	SettleWithBeneficiary(chainID int64, providerID identity.Identity, beneficiary common.Address, hermeses []common.Address) error
	SettleIntoStake(chainID int64, providerID identity.Identity, hermesID ...common.Address) error
	GetHermesFee(chainID int64, hermesID common.Address) (uint16, error)
	Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error
	CheckLatestWithdrawal(chainID int64, providerID identity.Identity, hermesID common.Address) (*big.Int, string, error)
	RetryWithdrawLatest(chainID int64, amountToWithdraw *big.Int, chid string, beneficiary common.Address, providerID identity.Identity) error
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
	hf                         hermesFees
	observerApi                observerApi
	// TODO: Consider adding chain ID to this as well.
	currentState map[identity.Identity]settlementState
	settleQueue  chan receivedPromise
	stop         chan struct{}
	once         sync.Once
}

type hermesFees struct {
	fees map[string]uint16
	m    sync.RWMutex
}

func (h *hermesFees) key(chainID int64, hermesID common.Address) string {
	return fmt.Sprint(chainID, hermesID.Hex())
}

func (h *hermesFees) set(chainID int64, hermesID common.Address, fee uint16) {
	h.m.Lock()
	defer h.m.Unlock()
	h.fees[h.key(chainID, hermesID)] = fee
}

func (h *hermesFees) get(chainID int64, hermesID common.Address) (uint16, bool) {
	h.m.RLock()
	defer h.m.RUnlock()
	got, ok := h.fees[h.key(chainID, hermesID)]
	return got, ok
}

// HermesPromiseSettlerConfig configures the hermes promise settler accordingly.
type HermesPromiseSettlerConfig struct {
	MaxFeeThreshold         float64
	MinAutoSettleAmount     float64
	MaxUnSettledAmount      float64
	L1ChainID               int64
	L2ChainID               int64
	SettlementCheckInterval time.Duration
	SettlementCheckTimeout  time.Duration
	BalanceThreshold        float64
}

var errFeeNotCovered = errors.New("fee not covered, cannot continue")

// NewHermesPromiseSettler creates a new instance of hermes promise settler.
func NewHermesPromiseSettler(transactor transactor, promiseStorage promiseStorage, paySettler paySettler, addressProvider addressProvider, hermesCallerFactory HermesCallerFactory, hermesURLGetter hermesURLGetter, channelProvider hermesChannelProvider, providerChannelStatusProvider providerChannelStatusProvider, registrationStatusProvider registrationStatusProvider, ks ks, settlementHistoryStorage settlementHistoryStorage, publisher eventbus.Publisher, observerApi observerApi, config HermesPromiseSettlerConfig) *hermesPromiseSettler {
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
		hf: hermesFees{
			fees: make(map[string]uint16),
		},
		observerApi: observerApi,
		// defaulting to a queue of 5, in case we have a few active identities.
		settleQueue: make(chan receivedPromise, 5),
		stop:        make(chan struct{}),
		transactor:  transactor,
	}
}

// GetHermesFee fetches the hermes fee.
func (aps *hermesPromiseSettler) GetHermesFee(chainID int64, hermesID common.Address) (uint16, error) {
	got, ok := aps.hf.get(chainID, hermesID)
	if !ok {
		fees, err := aps.bc.GetHermesFee(chainID, hermesID)
		if err != nil {
			return 0, err
		}

		aps.hf.set(chainID, hermesID, fees)
		return fees, nil
	}

	return got, nil
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
		registered:       true,
		settleInProgress: make(map[common.Address]struct{}),
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

	needs, maxFee := aps.needsSettling(s, aps.config.BalanceThreshold, aps.config.MaxFeeThreshold, aps.config.MinAutoSettleAmount, aps.config.MaxUnSettledAmount, channel, apep.Promise.ChainID)
	if needs {
		log.Info().Msgf("Starting auto settle for provider %v", id)
		aps.initiateSettling(channel, maxFee)
	}
}

func (aps *hermesPromiseSettler) initiateSettling(channel HermesChannel, maxFee *big.Int) {
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
		maxFee:      maxFee,
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
				p.maxFee,
			)
		}
	}
}

// SettleIntoStake settles the promise but transfers the money to stake increase, not to beneficiary.
func (aps *hermesPromiseSettler) SettleIntoStake(chainID int64, providerID identity.Identity, hermesIDs ...common.Address) error {
	for _, hermesID := range hermesIDs {
		channel, err := aps.channelProvider.Fetch(chainID, providerID, hermesID)
		if err != nil {
			log.Err(err).Fields(map[string]interface{}{
				"chain_id":  chainID,
				"provider":  providerID.Address,
				"hermes_id": hermesID,
			}).Msg("Failed to fetch a channel")
			return ErrNothingToSettle
		}

		hexR, err := hex.DecodeString(channel.lastPromise.R)
		if err != nil {
			return fmt.Errorf("could not decode R: %w", err)
		}
		channel.lastPromise.Promise.R = hexR
		err = aps.settle(
			func(promise crypto.Promise) (string, error) {
				return aps.transactor.SettleIntoStake(hermesID.Hex(), providerID.Address, promise)
			},
			providerID,
			hermesID,
			channel.lastPromise.Promise,
			channel.Beneficiary,
			channel.Channel.Settled,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// ErrNothingToSettle indicates that there is nothing to settle.
var ErrNothingToSettle = errors.New("nothing to settle for the given provider")

// ForceSettle forces the settlement for a provider
func (aps *hermesPromiseSettler) ForceSettle(chainID int64, providerID identity.Identity, hermesIDs ...common.Address) error {
	feeNotCoveredCount := 0
	for _, hermesID := range hermesIDs {
		channel, err := aps.channelProvider.Fetch(chainID, providerID, hermesID)
		if err != nil {
			log.Err(err).Fields(map[string]interface{}{
				"chain_id":  chainID,
				"provider":  providerID.Address,
				"hermes_id": hermesID,
			}).Msg("Failed to fetch a channel")
			return ErrNothingToSettle
		}

		hexR, err := hex.DecodeString(channel.lastPromise.R)
		if err != nil {
			return fmt.Errorf("could not decode R: %w", err)
		}

		channel.lastPromise.Promise.R = hexR
		err = aps.settle(
			func(promise crypto.Promise) (string, error) {
				return aps.transactor.SettleAndRebalance(hermesID.Hex(), providerID.Address, promise)
			},
			providerID,
			hermesID,
			channel.lastPromise.Promise,
			channel.Beneficiary,
			channel.Channel.Settled,
			nil,
		)
		if err != nil {
			if errors.Is(err, errFeeNotCovered) {
				log.Warn().Err(err).Str("hermes_id", hermesID.Hex()).Msg("fee not covered, skipping")
				feeNotCoveredCount++
				continue
			}

			return fmt.Errorf("settlements with hermes %q interrupted with an error: %w", hermesID.Hex(), err)
		}
	}

	if feeNotCoveredCount == len(hermesIDs) {
		return errors.New("fee not covered for all given hermeses, settled with none")
	}

	return nil
}

// ForceSettleInactiveHermeses forces the settlement for the inactive hermeses
func (aps *hermesPromiseSettler) ForceSettleInactiveHermeses(chainID int64, providerID identity.Identity) error {
	active := false
	approved := true
	inactiveHermeses, err := aps.observerApi.GetHermeses(&observer.HermesFilter{
		Active:   &active,
		Approved: &approved,
	})
	if err != nil {
		return fmt.Errorf("failed to get inactive hermeses: %w", err)
	}
	chainInactiveHermesesResponses, ok := inactiveHermeses[chainID]
	if !ok {
		log.Info().Msgf("no inactive hermeses found for chain: %d", chainID)
		return nil
	}

	chainInactiveHermeses := make([]common.Address, len(chainInactiveHermesesResponses))
	for i, h := range chainInactiveHermesesResponses {
		chainInactiveHermeses[i] = h.HermesAddress
	}

	return aps.ForceSettle(chainID, providerID, chainInactiveHermeses...)
}

// ForceSettle forces the settlement for a provider
func (aps *hermesPromiseSettler) SettleWithBeneficiary(chainID int64, providerID identity.Identity, beneficiary common.Address, hermeses []common.Address) error {
	var channel *HermesChannel = nil
	maxUnsettled := big.NewInt(0)
	for _, hermesID := range hermeses {
		hchannel, err := aps.channelProvider.Fetch(chainID, providerID, hermesID)
		if err != nil {
			log.Err(err).Fields(map[string]interface{}{
				"chain_id":  chainID,
				"provider":  providerID.Address,
				"hermes_id": hermesID,
			}).Msg("Failed to fetch a channel")
			return ErrNothingToSettle
		}

		if hchannel.lastPromise.Promise.Amount != nil {
			settled := hchannel.Channel.Settled
			if settled == nil {
				settled = big.NewInt(0)
			}
			unsettledAmount := new(big.Int).Sub(hchannel.lastPromise.Promise.Amount, settled)
			if unsettledAmount.Cmp(maxUnsettled) > 0 {
				maxUnsettled = unsettledAmount
				channel = &hchannel
			}
		}
	}

	if channel == nil {
		if len(hermeses) == 0 {
			return fmt.Errorf("cannot settle: no hermes provided")
		}
		if len(hermeses) == 1 {
			return fmt.Errorf("cannot settle: no unsettled funds for hermes: %s", hermeses[0].Hex())
		}
		return fmt.Errorf("cannot settle: no hermes with unsettled funds was found")
	}

	hexR, err := hex.DecodeString(channel.lastPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R: %w", err)
	}

	channel.lastPromise.Promise.R = hexR
	return aps.settle(
		func(promise crypto.Promise) (string, error) {
			return aps.transactor.SettleWithBeneficiary(providerID.Address, beneficiary.Hex(), channel.HermesID.Hex(), promise)
		},
		providerID,
		channel.HermesID,
		channel.lastPromise.Promise,
		beneficiary,
		channel.Channel.Settled,
		nil,
	)
}

// ErrSettleTimeout indicates that the settlement has timed out
var ErrSettleTimeout = errors.New("settle timeout")

func (aps *hermesPromiseSettler) updatePromiseWithLatestFee(hermesID common.Address, promise crypto.Promise, maxFee *big.Int) (crypto.Promise, error) {
	log.Debug().Msgf("Updating promise with latest fee. HermesID %v", hermesID.Hex())
	fees, err := aps.transactor.FetchSettleFees(promise.ChainID)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not fetch settle fees: %w", err)
	}

	if maxFee != nil && fees.Fee.Cmp(maxFee) == 1 {
		return crypto.Promise{}, fmt.Errorf("current fee is more than the max")
	}

	hermesCaller, err := aps.getHermesCaller(promise.ChainID, hermesID)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not fetch settle fees: %w", err)
	}

	updatedPromise, err := hermesCaller.UpdatePromiseFee(promise, fees.Fee)
	if err != nil {
		var hermesErr *HermesErrorResponse
		if errors.As(err, &hermesErr) {
			return crypto.Promise{}, fmt.Errorf("could not update promise fee: %w", err)
		}
		log.Err(err).Msg("could not update promise fee with unknown error, will try settling with outdated promise")
		return promise, nil
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
	if aps.isSettling(providerID, hermesID) {
		return errors.New("provider already has settlement in progress")
	}

	aps.setSettling(providerID, hermesID, true)
	log.Info().Msgf("Marked provider %v as requesting settlement", providerID)
	defer aps.setSettling(providerID, hermesID, false)

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
	channel, err := aps.addressProvider.GetChannelImplementationForHermes(fromChainID, hermesID)
	if err != nil {
		return err
	}

	consumerChannelAddress, err := aps.addressProvider.GetArbitraryChannelAddress(hermesID, registry, channel, providerID.ToCommonAddress())
	if err != nil {
		return fmt.Errorf("could not generate channel address: %w", err)
	}

	chid, err := crypto.GenerateProviderChannelIDForPayAndSettle(providerID.Address, hermesID.Hex())
	if err != nil {
		return fmt.Errorf("could not get channel id for pay and settle: %w", err)
	}

	// 1. calculate amount to withdraw - check balance on consumer channel
	data, err := aps.getHermesData(fromChainID, hermesID, providerID.ToCommonAddress())
	if err != nil {
		return err
	}

	if amountToWithdraw == nil {
		amountToWithdraw = data.Balance

		// TODO: Pull this from hermes contract in the future.
		maxWithdraw := crypto.FloatToBigMyst(500)
		if amountToWithdraw.Cmp(maxWithdraw) > 0 {
			amountToWithdraw = maxWithdraw
		}
	}

	err = aps.validateWithdrawalAmount(amountToWithdraw, toChainID)
	if err != nil {
		return err
	}

	currentPromise, err := aps.promiseStorage.Get(toChainID, chid)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	// 2. issue a self promise
	promisedAmount := data.LatestPromise.Amount
	if promisedAmount.Cmp(data.Settled) < 0 {
		// If consumer has an incorrect promise. Issue a correct one
		// together with a withdrawal request.
		promisedAmount = data.Settled
	}
	msg, err := aps.issueSelfPromise(fromChainID, toChainID, amountToWithdraw, promisedAmount, providerID, consumerChannelAddress, hermesID)
	if err != nil {
		return err
	}

	// 3. call hermes with the promise via the payandsettle endpoint
	ch := aps.paySettler.PayAndSettle(msg.Promise.R, *msg, providerID, "")
	err = <-ch
	if err != nil {
		log.Debug().Msgf("ERROR HERMES. provider:%s, hermes:%s, channel: %s", providerID, hermesID, channel)
		return fmt.Errorf("could not call hermes pay and settle:%w", err)
	}

	// 4. fetch the promise from storage
	latestPromise, err := aps.promiseStorage.Get(toChainID, chid)
	if err != nil {
		return err
	}

	if latestPromise.Promise.GetSignatureHexString() == currentPromise.Promise.GetSignatureHexString() {
		log.Warn().Msg("hermes promise was not updated, was not able to complete the withdrawal")
		return errors.New("promise was not updated, please request again")
	}

	decodedR, err := hex.DecodeString(latestPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R %w", err)
	}
	latestPromise.Promise.R = decodedR

	return aps.payAndSettleTransactor(toChainID, amountToWithdraw, beneficiary, providerID, chid, latestPromise, fromChainID)
}

func (aps *hermesPromiseSettler) CheckLatestWithdrawal(
	chainID int64,
	providerID identity.Identity,
	hermesID common.Address,
) (*big.Int, string, error) {
	chid, err := crypto.GenerateProviderChannelIDForPayAndSettle(providerID.Address, hermesID.Hex())
	if err != nil {
		return nil, "", fmt.Errorf("could not get channel id for pay and settle: %w", err)
	}

	latestPromise, err := aps.promiseStorage.Get(chainID, chid)
	if err != nil {
		return nil, chid, err
	}

	withdrawalChannel, err := aps.bc.GetProvidersWithdrawalChannel(chainID, hermesID, providerID.ToCommonAddress(), true)
	if err != nil {
		return nil, chid, err
	}

	return new(big.Int).Sub(latestPromise.Promise.Amount, withdrawalChannel.Settled), chid, nil
}

func (aps *hermesPromiseSettler) RetryWithdrawLatest(
	chainID int64,
	amountToWithdraw *big.Int,
	chid string,
	beneficiary common.Address,
	providerID identity.Identity,
) error {
	latestPromise, err := aps.promiseStorage.Get(chainID, chid)
	if err != nil {
		return err
	}

	decodedR, err := hex.DecodeString(latestPromise.R)
	if err != nil {
		return fmt.Errorf("could not decode R %w", err)
	}
	latestPromise.Promise.R = decodedR

	return aps.payAndSettleTransactor(chainID, amountToWithdraw, beneficiary, providerID, chid, latestPromise, chainID)

}

func (aps *hermesPromiseSettler) payAndSettleTransactor(toChainID int64, amountToWithdraw *big.Int, beneficiary common.Address, providerID identity.Identity, chid string, promiseFromStorage HermesPromise, fromChain int64) error {
	decodedR, err := hex.DecodeString(promiseFromStorage.R)
	if err != nil {
		return fmt.Errorf("could not decode R %w", err)
	}
	promiseFromStorage.Promise.R = decodedR

	// 5. add the missing beneficiary signature
	payload := crypto.NewPayAndSettleBeneficiaryPayload(beneficiary, toChainID, chid, promiseFromStorage.Promise.Amount, client.ToBytes32(promiseFromStorage.Promise.R))
	err = payload.Sign(aps.ks, providerID.ToCommonAddress())
	if err != nil {
		return fmt.Errorf("could not sign pay and settle payload: %w", err)
	}

	settleFunc := func(promise crypto.Promise) (string, error) {
		id, err := aps.transactor.PayAndSettle(promiseFromStorage.HermesID.Hex(), providerID.Address, promise, payload.Beneficiary.Hex(), hex.EncodeToString(payload.Signature))
		if err != nil {
			return "", err
		}

		aps.publisher.Publish(event.AppTopicWithdrawalRequested, event.AppEventWithdrawalRequested{
			ProviderID: providerID,
			HermesID:   promiseFromStorage.HermesID,
			FromChain:  fromChain,
			ToChain:    promiseFromStorage.Promise.ChainID,
		})

		return id, nil
	}

	go func() {
		log.Info().Msg("calling transactor")
		err := aps.payAndSettle(
			settleFunc,
			providerID,
			promiseFromStorage.HermesID,
			promiseFromStorage.Promise,
			beneficiary,
			amountToWithdraw,
			promiseFromStorage,
		)
		if err != nil {
			log.Err(err).Msg("could not withdraw, failed to call transactor")
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

func (aps *hermesPromiseSettler) validateWithdrawalAmount(amount *big.Int, toChain int64) error {
	fees, err := aps.transactor.FetchSettleFees(toChain)
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

	id, err := aps.settlePayAndSettleWithRetry(settleFunc, withdrawalAmount, promiseFromStorage)
	if err != nil {
		return err
	}

	channelID, err := crypto.GenerateProviderChannelIDForPayAndSettle(provider.Address, hermesID.Hex())
	if err != nil {
		return fmt.Errorf("could not generate provider channel address: %w", err)
	}

	errCh := aps.listenForSettlement(hermesID, beneficiary, promise, provider, aps.toBytes32(channelID), id, true)
	return <-errCh
}

func payAndSettleErrorShouldRetry(err error) bool {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == 409 || apiErr.Status >= 500
	}

	return false
}

func (aps *hermesPromiseSettler) settlePayAndSettleWithRetry(
	settleFunc func(promise crypto.Promise) (string, error),
	withdrawalAmount *big.Int,
	promiseFromStorage HermesPromise,
) (string, error) {
	promise, err := aps.updatePromiseWithLatestFee(promiseFromStorage.HermesID, promiseFromStorage.Promise, nil)
	if err != nil {
		log.Error().Err(err).Msg("Could not update promise fee")
		return "", err
	}

	if promise.Fee.Cmp(withdrawalAmount) > 0 {
		log.Error().Fields(map[string]interface{}{
			"promiseAmount": promise.Amount.String(),
			"transactorFee": promise.Fee.String(),
		}).Err(err).Msg("Earned amount too small for withdrawal")
		return "", fmt.Errorf("amount too small for settlement. Need at least %v, have %v", promise.Fee.String(), withdrawalAmount.String())
	}

	id, err := settleFunc(promise)
	if err == nil {
		return id, nil
	}

	if err != nil && !payAndSettleErrorShouldRetry(err) {
		log.Err(err).Msg("tried to settle withdrawal but failed")
		return "", err
	}

	log.Warn().Err(err).Msg("got an error for which we can retry a withdrawal, will do that")
	// Retry 10 times incase transactor failed to accept our settlement request.
	// There is no point in going for longer than 5 minutes, after that fees
	// will expire and there will be a different amount of fees to pay meaning
	// user would get a different amount of money after the withdrawal.
	// It would be rather strange from users perspective if he ends up paying
	// way more than he agreed when creating the request.
	for i := 0; i < 10; i++ {
		select {
		case <-time.After(time.Second * 30):
			log.Info().Int("count", i+1).Msg("retrying a call to settle withdrawal")

			id, err := settleFunc(promise)
			if err != nil {
				if payAndSettleErrorShouldRetry(err) {
					log.Warn().Err(err).Msg("failed when retrying withdrawal")
					continue
				}
				return "", err
			}

			return id, nil
		case <-aps.stop:
			return "", errors.New("stopped trying to withdraw, will not finish")
		}
	}

	return "", errors.New("out of retries, transactor never accepted our request to pay and settle")
}

func (aps *hermesPromiseSettler) issueSelfPromise(fromChain, toChain int64, amount, previousPromiseAmount *big.Int, providerID identity.Identity, consumerChannelAddress, hermesAddress common.Address) (*crypto.ExchangeMessage, error) {
	r := aps.generateR()
	agreementID := aps.generateAgreementID()
	invoice := crypto.CreateInvoice(agreementID, amount, big.NewInt(0), r, toChain)
	invoice.Provider = providerID.ToCommonAddress().Hex()

	promise, err := crypto.CreatePromise(consumerChannelAddress.Hex(), fromChain, big.NewInt(0).Add(amount, previousPromiseAmount), big.NewInt(0), invoice.Hashlock, aps.ks, providerID.ToCommonAddress())
	if err != nil {
		return nil, fmt.Errorf("could not create promise: %w", err)
	}

	promise.R = r

	msg, err := crypto.CreateExchangeMessageWithPromise(toChain, invoice, promise, hermesAddress.Hex(), aps.ks, providerID.ToCommonAddress())
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

func (aps *hermesPromiseSettler) getHermesData(chainID int64, hermesID, id common.Address) (*HermesUserInfo, error) {
	caller, err := aps.getHermesCaller(chainID, hermesID)
	if err != nil {
		return nil, err
	}

	data, err := caller.GetConsumerData(chainID, id.Hex())
	if err != nil {
		return nil, err
	}

	if data.Balance.Cmp(big.NewInt(0)) > 0 {
		return data.fillZerosIfBigIntNull(), nil
	}

	// if hermes returned zero, re-check with BC
	myst, err := aps.addressProvider.GetMystAddress(chainID)
	if err != nil {
		return nil, err
	}

	channelAddress, err := aps.addressProvider.GetActiveChannelAddress(chainID, id)
	if err != nil {
		return nil, err
	}

	balance, err := aps.bc.GetMystBalance(chainID, myst, channelAddress)
	if err != nil {
		return nil, err
	}

	if balance.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("nothing to withdraw. Balance in channel %v is %v", data.ChannelID, data.Balance)
	}

	data.Balance = balance

	return data.fillZerosIfBigIntNull(), nil
}

func (aps *hermesPromiseSettler) settle(
	settleFunc func(promise crypto.Promise) (string, error),
	provider identity.Identity,
	hermesID common.Address,
	promise crypto.Promise,
	beneficiary common.Address,
	settled *big.Int,
	maxFee *big.Int,
) error {
	if aps.isSettling(provider, hermesID) {
		return errors.New("provider already has settlement in progress")
	}
	aps.setSettling(provider, hermesID, true)
	defer aps.setSettling(provider, hermesID, false)

	log.Info().Msgf("Marked provider %v as requesting settlement", provider)

	updatedPromise, err := aps.updatePromiseWithLatestFee(hermesID, promise, maxFee)
	if err != nil {
		log.Error().Err(err).Msg("Could not update promise fee")
		return err
	}

	if settled == nil {
		settled = new(big.Int)
	}

	amountToSettle := new(big.Int).Sub(updatedPromise.Amount, settled)
	if amountToSettle.Cmp(big.NewInt(0)) <= 0 {
		log.Warn().Msgf("Tried to settle for %s MYST", amountToSettle.String())
		return nil
	}

	fee, err := aps.bc.CalculateHermesFee(promise.ChainID, hermesID, amountToSettle)
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate hermes fee")
		return err
	}

	totalFees := new(big.Int).Add(fee, updatedPromise.Fee)
	if totalFees.Cmp(amountToSettle) > 0 {
		log.Error().Fields(map[string]interface{}{
			"amountToSettle": amountToSettle.String(),
			"promiseAmount":  updatedPromise.Amount.String(),
			"settled":        settled.String(),
			"transactorFee":  updatedPromise.Fee.String(),
			"hermesFee":      fee.String(),
			"totalFees":      totalFees.String(),
		}).Err(err).Msg("Earned amount too small for settling")
		return fmt.Errorf("settlement fees exceed earning amount. Please provide more service and try again. Current earnings: %v, current fees: %v: %w", amountToSettle, totalFees, errFeeNotCovered)
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

	errCh := aps.listenForSettlement(hermesID, beneficiary, updatedPromise, provider, aps.toBytes32(channelID), id, false)
	return <-errCh
}

func (aps *hermesPromiseSettler) listenForSettlement(hermesID, beneficiary common.Address, promise crypto.Promise, provider identity.Identity, providerChannelID [32]byte, queueID string, isWithdrawal bool) <-chan error {
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
					she := SettlementHistoryEntry{
						TxHash:           common.HexToHash(""),
						BlockExplorerURL: "",
						ProviderID:       provider,
						HermesID:         hermesID,
						// TODO: this should probably be either provider channel address or the consumer address from the promise, not truncated provider channel address.
						ChannelAddress: common.BytesToAddress(providerChannelID[:]),
						Time:           time.Now().UTC(),
						Promise:        promise,
						Beneficiary:    beneficiary,
						Error:          res.Error,
						IsWithdrawal:   isWithdrawal,
					}
					err = aps.settlementHistoryStorage.Store(she)
					if err != nil {
						log.Error().Err(err).Msg("Could not store settlement history")
					}
					errCh <- fmt.Errorf("transactor reported queue error for id %v: %v", queueID, res.Error)
					return
				}

				// at this point, state should be done. If it is something else, abort.
				if state != "done" {
					errCh <- fmt.Errorf("transactor reported unknown settlement state for id %v, state %v", queueID, state)
					return
				}

				filtered, err := aps.findSettlementInBCLogs(promise.ChainID, hermesID, providerChannelID, aps.toBytes32(common.Bytes2Hex(promise.R)))
				if err != nil {
					errCh <- fmt.Errorf("could not get settlement event from bc: %w", err)
					return
				}

				if len(filtered) == 0 {
					log.Warn().Fields(map[string]interface{}{
						"hermesID": hermesID.Hex(),
						"provider": provider.Address,
						"queueID":  queueID,
					}).Err(err).Msg("no settlement found, will try again later")
					break
				}

				ch, err := aps.channelProvider.Fetch(promise.ChainID, provider, hermesID)
				if err != nil {
					log.Error().Err(err).Msgf("Resync failed for provider %v", provider)
				} else {
					log.Info().Msgf("Resync success for provider %v", provider)
				}

				for _, info := range filtered {
					uri, err := formTXUrl(info.Raw.TxHash.Hex(), promise.ChainID)
					if err != nil {
						log.Err(err).Msg("could not generate tx uri")
					}

					she := SettlementHistoryEntry{
						TxHash:           info.Raw.TxHash,
						BlockExplorerURL: uri,
						ProviderID:       provider,
						HermesID:         hermesID,
						// TODO: this should probably be either provider channel address or the consumer address from the promise, not truncated provider channel address.
						ChannelAddress: common.BytesToAddress(providerChannelID[:]),
						Time:           time.Now().UTC(),
						Promise:        promise,
						Beneficiary:    beneficiary,
						Amount:         info.AmountSentToBeneficiary,
						Fees:           info.Fees,
						TotalSettled:   ch.Channel.Settled,
						Error:          aps.generateSettlementErrorMsg(promise.ChainID, info.Raw.TxHash),
						IsWithdrawal:   isWithdrawal,
					}

					err = aps.settlementHistoryStorage.Store(she)
					if err != nil {
						log.Error().Err(err).Msg("Could not store settlement history")
					}

					log.Debug().Str("tx_hash", info.Raw.TxHash.Hex()).Msg("saved a settlement")
				}

				aps.publisher.Publish(event.AppTopicSettlementComplete, event.AppEventSettlementComplete{
					ProviderID: provider,
					HermesID:   hermesID,
					ChainID:    promise.ChainID,
				})
				log.Info().Msgf("Settling complete for provider %v", provider)
				return
			}
		}
	}()
	return errCh
}

func (aps *hermesPromiseSettler) generateSettlementErrorMsg(chainID int64, hash common.Hash) string {
	receipt, err := aps.bc.TransactionReceipt(chainID, hash)
	if err != nil {
		log.Err(err).Msg("failed to get receipt for settlemnt")
		return "Could not get receipt"
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return "Transaction was reverted"
	}

	return ""
}

func (aps *hermesPromiseSettler) toBytes32(hexValue string) [32]byte {
	var arr [32]byte
	copy(arr[:], crypto.HexToBytes(hexValue)[:32])
	return arr
}

var errNoSettlementFound = errors.New("no settlement found")

func (aps *hermesPromiseSettler) findSettlementInBCLogs(chainID int64, hermesID common.Address, providerAddress, r [32]byte) ([]bindings.HermesImplementationPromiseSettled, error) {
	latest, err := aps.bc.HeaderByNumber(chainID, nil)
	if err != nil {
		return nil, err
	}
	blockNo := latest.Number.Uint64()
	from := aps.safeSub(blockNo, 800)

	filtered, err := aps.bc.FilterPromiseSettledEventByChannelID(chainID, from, nil, hermesID, [][32]byte{providerAddress})
	if err != nil {
		return nil, err
	}

	match := func(v bindings.HermesImplementationPromiseSettled) bool {
		if !bytes.EqualFold(v.ChannelId[:], providerAddress[:]) {
			return false
		}
		if !bytes.EqualFold(v.Lock[:], r[:]) {
			return false
		}

		return true
	}

	matched := make([]bindings.HermesImplementationPromiseSettled, 0)
	for _, v := range filtered {
		log.Info().Str("expected", hex.EncodeToString(providerAddress[:])).Str("got", hex.EncodeToString(v.ChannelId[:])).Msg("filtering")

		if match(v) {
			matched = append(matched, v)
			log.Debug().Str("tx_hash", v.Raw.TxHash.Hex()).Msg("matched a settlement")
		}
	}

	return matched, nil
}

func (aps *hermesPromiseSettler) safeSub(a uint64, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}

func (aps *hermesPromiseSettler) isSettling(id identity.Identity, hermesID common.Address) bool {
	aps.lock.RLock()
	defer aps.lock.RUnlock()
	v, ok := aps.currentState[id]
	if !ok {
		return false
	}

	if v.settleInProgress == nil {
		v.settleInProgress = make(map[common.Address]struct{})
		aps.currentState[id] = v
	}

	_, ok = v.settleInProgress[hermesID]

	return ok
}

func (aps *hermesPromiseSettler) setSettling(id identity.Identity, hermesID common.Address, settling bool) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	v := aps.currentState[id]

	if v.settleInProgress == nil {
		v.settleInProgress = make(map[common.Address]struct{})
	}

	if settling {
		v.settleInProgress[hermesID] = struct{}{}
	} else {
		delete(v.settleInProgress, hermesID)
	}

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
			err = aps.ForceSettleInactiveHermeses(aps.chainID(), addr)
			if err != nil {
				log.Error().Err(err).Msgf("could not settle inactive hermeses %v", addr)
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
	registered bool

	settleInProgress map[common.Address]struct{}
}

func (aps *hermesPromiseSettler) needsSettling(ss settlementState, balanceThreshold float64, feeThreshold float64, minSettleAmount, maxUnSettledAmount float64, channel HermesChannel, chainID int64) (bool, *big.Int) {
	if !ss.registered {
		return false, nil
	}

	if _, ok := ss.settleInProgress[channel.HermesID]; ok {
		return false, nil
	}

	if channel.Channel.Stake.Cmp(big.NewInt(0)) == 0 {
		// no stake mode
		unsettledAmount := channel.UnsettledBalance()
		if unsettledAmount.Cmp(crypto.FloatToBigMyst(maxUnSettledAmount)) > 0 {
			return true, nil
		}
		if unsettledAmount.Cmp(crypto.FloatToBigMyst(minSettleAmount)) >= 0 {
			settleFees, err := aps.transactor.FetchSettleFees(chainID)
			if err != nil {
				log.Err(err).Msgf("will not use settlement fees to check if settling is needed")
				return false, nil
			}
			//set max fee to 10% more than current
			maxFee, _ := new(big.Float).Mul(new(big.Float).SetInt(settleFees.Fee), big.NewFloat(1.1)).Int(nil)
			unsettledBalance := new(big.Float).SetInt(channel.UnsettledBalance())
			calculatedFeesThreshold := new(big.Float).Mul(big.NewFloat(feeThreshold), unsettledBalance)
			calculatedFeesThresholdInt, _ := calculatedFeesThreshold.Int(nil)
			return settleFees.Fee.Cmp(calculatedFeesThresholdInt) < 0, maxFee
		}
		return false, nil
	}

	floated := new(big.Float).SetInt(channel.availableBalance())
	calculatedThreshold := new(big.Float).Mul(big.NewFloat(balanceThreshold), floated)
	possibleEarnings := channel.UnsettledBalance()
	i, _ := calculatedThreshold.Int(nil)
	if possibleEarnings.Cmp(i) == -1 {
		return false, nil
	}

	if channel.balance().Cmp(i) <= 0 {
		return true, nil
	}

	return false, nil
}

func formTXUrl(txHash string, chainID int64) (string, error) {
	if len(txHash) == 0 {
		return "", nil
	}

	if txHash == common.HexToHash("").Hex() {
		return "", nil
	}

	switch chainID {
	case 1:
		return fmt.Sprintf("https://etherscan.io/tx/%v", txHash), nil
	case 5:
		return fmt.Sprintf("https://goerli.etherscan.io/tx/%v", txHash), nil
	case 80001:
		return fmt.Sprintf("https://mumbai.polygonscan.com/tx/%v", txHash), nil
	case 137:
		return fmt.Sprintf("https://polygonscan.com/tx/%v", txHash), nil
	default:
		return "", fmt.Errorf("unsupported chainID(%v) for tx url", chainID)
	}
}
