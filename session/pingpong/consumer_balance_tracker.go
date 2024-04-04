/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	pevent "github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/units"
)

type balanceKey string

func newBalanceKey(chainID int64, id identity.Identity) balanceKey {
	return balanceKey(fmt.Sprintf("%v_%v", id.Address, chainID))
}

type balances struct {
	sync.Mutex
	valuesMap map[balanceKey]ConsumerBalance
}

type transactorBounties struct {
	sync.Mutex
	valuesMap map[balanceKey]*big.Int
}

// ConsumerBalanceTracker keeps track of consumer balances.
// TODO: this needs to take into account the saved state.
type ConsumerBalanceTracker struct {
	balances           balances
	transactorBounties transactorBounties

	addressProvider                      addressProvider
	registry                             registrationStatusProvider
	consumerBalanceChecker               consumerBalanceChecker
	bus                                  eventbus.EventBus
	consumerGrandTotalsStorage           consumerTotalsStorage
	consumerInfoGetter                   consumerInfoGetter
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider
	blockchainInfoProvider               blockchainInfoProvider
	stop                                 chan struct{}
	once                                 sync.Once

	fullBalanceUpdateThrottle map[string]struct{}
	fullBalanceUpdateLock     sync.Mutex
	balanceSyncer             *balanceSyncer

	cfg ConsumerBalanceTrackerConfig
}

type transactorRegistrationStatusProvider interface {
	FetchRegistrationFees(chainID int64) (registry.FeesResponse, error)
	FetchRegistrationStatus(id string) ([]registry.TransactorStatusResponse, error)
}

type blockchainInfoProvider interface {
	GetConsumerChannelsHermes(chainID int64, channelAddress common.Address) (client.ConsumersHermes, error)
}

// PollConfig sets the interval and timeout for polling.
type PollConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// ConsumerBalanceTrackerConfig represents the consumer balance tracker configuration.
type ConsumerBalanceTrackerConfig struct {
	FastSync PollConfig
	LongSync PollConfig
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(
	publisher eventbus.EventBus,
	consumerBalanceChecker consumerBalanceChecker,
	consumerGrandTotalsStorage consumerTotalsStorage,
	consumerInfoGetter consumerInfoGetter,
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider,
	registry registrationStatusProvider,
	addressProvider addressProvider,
	blockchainInfoProvider blockchainInfoProvider,
	cfg ConsumerBalanceTrackerConfig,
) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                             balances{valuesMap: make(map[balanceKey]ConsumerBalance)},
		transactorBounties:                   transactorBounties{valuesMap: make(map[balanceKey]*big.Int)},
		consumerBalanceChecker:               consumerBalanceChecker,
		bus:                                  publisher,
		consumerGrandTotalsStorage:           consumerGrandTotalsStorage,
		consumerInfoGetter:                   consumerInfoGetter,
		transactorRegistrationStatusProvider: transactorRegistrationStatusProvider,
		blockchainInfoProvider:               blockchainInfoProvider,
		registry:                             registry,
		addressProvider:                      addressProvider,
		stop:                                 make(chan struct{}),
		cfg:                                  cfg,
		fullBalanceUpdateThrottle:            make(map[string]struct{}),
		balanceSyncer:                        newBalanceSyncer(),
	}
}

type consumerInfoGetter interface {
	GetConsumerData(chainID int64, id string, cacheDuration time.Duration) (HermesUserInfo, error)
}

type consumerBalanceChecker interface {
	GetConsumerChannel(chainID int64, addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error)
	GetMystBalance(chainID int64, mystAddress, identity common.Address) (*big.Int, error)
}

var errBalanceNotOffchain = errors.New("balance is not offchain, can't use hermes to check")

// Subscribe subscribes the consumer balance tracker to relevant events
func (cbt *ConsumerBalanceTracker) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(registry.AppTopicIdentityRegistration, cbt.handleRegistrationEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(string(nodevent.StatusStopped), cbt.handleStopEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(event.AppTopicGrandTotalChanged, cbt.handleGrandTotalChanged)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(pevent.AppTopicOrderUpdated, cbt.handleOrderUpdatedEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(event.AppTopicSettlementComplete, cbt.handleSettlementComplete)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(event.AppTopicWithdrawalRequested, cbt.handleWithdrawalRequested)
	if err != nil {
		return err
	}
	return bus.SubscribeAsync(identity.AppTopicIdentityUnlock, cbt.handleUnlockEvent)
}

// settlements increase balance on the chain they are settled on.
func (cbt *ConsumerBalanceTracker) handleSettlementComplete(ev event.AppEventSettlementComplete) {
	go cbt.aggressiveSync(ev.ChainID, ev.ProviderID, cbt.cfg.FastSync.Timeout, cbt.cfg.FastSync.Interval)
}

// withdrawals decrease balance on the from chain.
func (cbt *ConsumerBalanceTracker) handleWithdrawalRequested(ev event.AppEventWithdrawalRequested) {
	go cbt.aggressiveSync(ev.FromChain, ev.ProviderID, cbt.cfg.FastSync.Timeout, cbt.cfg.FastSync.Interval)
}

func (cbt *ConsumerBalanceTracker) handleOrderUpdatedEvent(ev pevent.AppEventOrderUpdated) {
	if !ev.Status.Paid() {
		return
	}

	go cbt.aggressiveSync(config.GetInt64(config.FlagChainID), identity.FromAddress(ev.IdentityAddress), cbt.cfg.FastSync.Timeout, cbt.cfg.FastSync.Interval)
}

// Performs a more aggresive type of sync on BC for the given identity on the given chain.
// Should be used after events that cause a state change on blockchain.
func (cbt *ConsumerBalanceTracker) aggressiveSync(chainID int64, id identity.Identity, timeout, frequency time.Duration) {
	b, ok := cbt.getBalance(chainID, id)
	if ok && b.IsOffchain {
		log.Info().Bool("is_offchain", b.IsOffchain).Msg("skipping aggresive sync")
		return
	}

	cbt.startJob(chainID, id, timeout, frequency)
}

func (cbt *ConsumerBalanceTracker) formJobSyncKey(chainID int64, id identity.Identity, timeout, frequency time.Duration) string {
	return fmt.Sprintf("%v%v%v%v", chainID, id.ToCommonAddress().Hex(), timeout, frequency)
}

// NeedsForceSync returns true if balance needs to be force synced.
func (cbt *ConsumerBalanceTracker) NeedsForceSync(chainID int64, id identity.Identity) bool {
	v, ok := cbt.getBalance(chainID, id)
	if !ok {
		return true
	}

	// Offchain balances expire after configured amount of time and need to be resynced.
	if v.OffchainNeedsSync() {
		return true
	}

	// Balance doesn't always go to 0 but connections can still fail.
	if v.BCBalance.Cmp(units.SingleGweiInWei()) < 0 {
		return true
	}

	return false
}

// GetBalance gets the current balance for given identity
func (cbt *ConsumerBalanceTracker) GetBalance(chainID int64, id identity.Identity) *big.Int {
	if v, ok := cbt.getBalance(chainID, id); ok {
		return v.GetBalance()
	}
	return new(big.Int)
}

func (cbt *ConsumerBalanceTracker) publishChangeEvent(id identity.Identity, before, after *big.Int) {
	if before == nil || after == nil || before.Cmp(after) == 0 {
		return
	}

	cbt.bus.Publish(event.AppTopicBalanceChanged, event.AppEventBalanceChanged{
		Identity: id,
		Previous: before,
		Current:  after,
	})
}

func (cbt *ConsumerBalanceTracker) handleUnlockEvent(data identity.AppEventIdentityUnlock) {
	err := cbt.recoverGrandTotalPromised(data.ChainID, data.ID)
	if err != nil {
		log.Error().Err(err).Msg("Could not recover Grand Total Promised")
	}

	status, err := cbt.registry.GetRegistrationStatus(data.ChainID, data.ID)
	if err != nil {
		log.Error().Err(err).Msg("Could not recover get registration status")
	}

	switch status {
	case registry.InProgress:
		cbt.alignWithTransactor(data.ChainID, data.ID)
	default:
		cbt.ForceBalanceUpdate(data.ChainID, data.ID)
	}

	go cbt.lifetimeBCSync(data.ChainID, data.ID)
}

func (cbt *ConsumerBalanceTracker) handleGrandTotalChanged(ev event.AppEventGrandTotalChanged) {
	if _, ok := cbt.getBalance(ev.ChainID, ev.ConsumerID); !ok {
		cbt.ForceBalanceUpdate(ev.ChainID, ev.ConsumerID)
		return
	}

	cbt.updateGrandTotal(ev.ChainID, ev.ConsumerID, ev.Current)
}

func (cbt *ConsumerBalanceTracker) getUnregisteredChannelBalance(chainID int64, id identity.Identity) (*big.Int, error) {
	addr, err := cbt.addressProvider.GetActiveChannelAddress(chainID, id.ToCommonAddress())
	if err != nil {
		return new(big.Int), err
	}

	myst, err := cbt.addressProvider.GetMystAddress(chainID)
	if err != nil {
		return new(big.Int), err
	}

	balance, err := cbt.consumerBalanceChecker.GetMystBalance(chainID, myst, addr)
	if err != nil {
		return new(big.Int), err
	}
	return balance, nil
}

func (cbt *ConsumerBalanceTracker) lifetimeBCSync(chainID int64, id identity.Identity) {
	b, ok := cbt.getBalance(chainID, id)
	if ok && b.IsOffchain {
		log.Info().Bool("is_offchain", b.IsOffchain).Msg("skipping external channel top-up tracking")
		return
	}

	// 100 years should be close enough to never
	timeout := time.Hour * 24 * 365 * 100
	cbt.startJob(chainID, id, timeout, cbt.cfg.LongSync.Interval)
}

func (cbt *ConsumerBalanceTracker) startJob(chainID int64, id identity.Identity, timeout, frequency time.Duration) {
	job, exists := cbt.balanceSyncer.PeriodiclySyncBalance(
		cbt.formJobSyncKey(chainID, id, timeout, frequency),
		func(stop <-chan struct{}) {
			cbt.periodicSync(stop, chainID, id, frequency)
		},
		timeout,
	)

	if exists {
		return
	}

	go func() {
		select {
		case <-cbt.stop:
			job.Stop()
			return
		case <-job.Done():
			return
		}
	}()
}

func (cbt *ConsumerBalanceTracker) periodicSync(stop <-chan struct{}, chainID int64, id identity.Identity, syncPeriod time.Duration) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(syncPeriod):
			_ = cbt.ForceBalanceUpdate(chainID, id)
		}
	}
}

func (cbt *ConsumerBalanceTracker) alignWithHermes(chainID int64, id identity.Identity) (*big.Int, *big.Int, error) {
	var boff backoff.BackOff
	eback := backoff.NewExponentialBackOff()
	eback.MaxElapsedTime = time.Second * 15
	eback.InitialInterval = time.Second * 1

	boff = backoff.WithMaxRetries(eback, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	go func() {
		select {
		case <-cbt.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	boff = backoff.WithContext(boff, ctx)
	balance := cbt.GetBalance(chainID, id)
	promised := new(big.Int)
	alignBalance := func() error {
		consumer, err := cbt.consumerInfoGetter.GetConsumerData(chainID, id.Address, 5*time.Second)
		if err != nil {
			var syntax *json.SyntaxError
			if errors.As(err, &syntax) {
				cancel()
				log.Err(err).Msg("hermes response is malformed JSON can't check if offchain")
				return err
			}

			if errors.Is(err, ErrHermesNotFound) {
				// Hermes doesn't know about this identity meaning it's not offchain. Cancel.
				cancel()
				return errBalanceNotOffchain
			}

			return err
		}
		if !consumer.IsOffchain {
			// Hermes knows about this identity, but it's not offchain. Cancel.
			cancel()
			return errBalanceNotOffchain
		}

		if consumer.LatestPromise.Amount != nil {
			promised = consumer.LatestPromise.Amount
		}

		if isSettledBiggerThanPromised(consumer.Settled, promised) {
			promised, err = cbt.getPromisedWhenSettledIsBigger(consumer, promised, chainID, id.ToCommonAddress())
			if err != nil {
				return err
			}
		}

		previous, _ := cbt.getBalance(chainID, id)
		cbt.setBalance(chainID, id, ConsumerBalance{
			BCBalance:          consumer.Balance,
			BCSettled:          consumer.Settled,
			GrandTotalPromised: promised,
			IsOffchain:         true,
			LastOffchainSync:   time.Now().UTC(),
		})

		currentBalance, _ := cbt.getBalance(chainID, id)
		go cbt.publishChangeEvent(id, previous.GetBalance(), currentBalance.GetBalance())
		balance = consumer.Balance
		return nil
	}

	return balance, promised, backoff.Retry(alignBalance, boff)
}

// ForceBalanceUpdateCached forces a balance update for the given identity only if the last call to this func was done no sooner than a minute ago.
func (cbt *ConsumerBalanceTracker) ForceBalanceUpdateCached(chainID int64, id identity.Identity) *big.Int {
	cbt.fullBalanceUpdateLock.Lock()
	defer cbt.fullBalanceUpdateLock.Unlock()

	key := getKeyForForceBalanceCache(chainID, id)
	_, ok := cbt.fullBalanceUpdateThrottle[key]
	if ok {
		return cbt.GetBalance(chainID, id)
	}

	currentBalance := cbt.ForceBalanceUpdate(chainID, id)
	cbt.fullBalanceUpdateThrottle[key] = struct{}{}

	go func() {
		select {
		case <-time.After(time.Minute):
			cbt.deleteCachedForceBalance(key)
		case <-cbt.stop:
			return
		}
	}()

	return currentBalance
}

func (cbt *ConsumerBalanceTracker) deleteCachedForceBalance(key string) {
	cbt.fullBalanceUpdateLock.Lock()
	defer cbt.fullBalanceUpdateLock.Unlock()

	delete(cbt.fullBalanceUpdateThrottle, key)
}

func getKeyForForceBalanceCache(chainID int64, id identity.Identity) string {
	return fmt.Sprintf("%v_%v", id.ToCommonAddress().Hex(), chainID)
}

// ForceBalanceUpdate forces a balance update and returns the updated balance
func (cbt *ConsumerBalanceTracker) ForceBalanceUpdate(chainID int64, id identity.Identity) *big.Int {
	fallback, ok := cbt.getBalance(chainID, id)
	if !ok {
		fallback.BCBalance = big.NewInt(0)
	}

	addr, err := cbt.addressProvider.GetActiveChannelAddress(chainID, id.ToCommonAddress())
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate channel address")
		return fallback.BCBalance
	}

	myst, err := cbt.addressProvider.GetMystAddress(chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not get myst address")
		return new(big.Int)
	}

	balance, lastPromised, err := cbt.alignWithHermes(chainID, id)
	if err != nil {
		if !errors.Is(err, errBalanceNotOffchain) {
			log.Error().Err(err).Msg("align with hermes failed with a critical error, offchain balance out of sync")
		}
		if !errors.Is(err, errBalanceNotOffchain) && fallback.IsOffchain {
			log.Warn().Msg("offchain sync failed but found a cache entry, will return that")
			return fallback.BCBalance
		}
	} else {
		return balance
	}

	cc, err := cbt.consumerBalanceChecker.GetConsumerChannel(chainID, addr, myst)
	if err != nil {
		// This indicates we're not registered, check for transactor bounty first and then unregistered balance.
		log.Warn().Err(err).Msg("Could not get consumer channel")
		if client.IsErrConnectionFailed(err) {
			log.Debug().Msg("tried to get consumer channel and got a connection error, will return last known balance")
			return fallback.BCBalance
		}

		var unregisteredBalance *big.Int
		// If registration is in progress, check transactor for bounty amount.
		bountyAmount, ok := cbt.getTransactorBounty(chainID, id)
		if ok {
			// if bounty from transactor is 0 it will be the unregistered balance of the channel.
			unregisteredBalance = bountyAmount
		} else {
			// If error was not for connection it indicates we're not registered, check for unregistered balance.
			unregisteredBalance, err = cbt.getUnregisteredChannelBalance(chainID, id)
			if err != nil {
				log.Error().Err(err).Msg("could not get unregistered balance")
				return fallback.BCBalance
			}
		}

		cbt.setBalance(chainID, id, ConsumerBalance{
			BCBalance:          unregisteredBalance,
			BCSettled:          new(big.Int),
			GrandTotalPromised: new(big.Int),
		})

		currentBalance, _ := cbt.getBalance(chainID, id)
		go cbt.publishChangeEvent(id, new(big.Int), currentBalance.GetBalance())
		return unregisteredBalance
	}

	hermes, err := cbt.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not get active hermes address")
		return fallback.BCBalance
	}

	grandTotal, err := cbt.consumerGrandTotalsStorage.Get(chainID, id, hermes)
	if errors.Is(err, ErrNotFound) || (err == nil && lastPromised != nil && grandTotal.Cmp(lastPromised) == -1) {
		err := cbt.consumerGrandTotalsStorage.Store(chainID, id, hermes, lastPromised)
		if err != nil {
			log.Error().Err(err).Msg("Could not recover Grand Total Promised")
		}
		grandTotal = lastPromised
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Error().Err(err).Msg("Could not get consumer grand total promised")
		return fallback.BCBalance
	}

	before := new(big.Int)
	if v, ok := cbt.getBalance(chainID, id); ok {
		before = v.GetBalance()
	}

	cbt.setBalance(chainID, id, ConsumerBalance{
		BCBalance:          cc.Balance,
		BCSettled:          cc.Settled,
		GrandTotalPromised: grandTotal,
	})

	currentBalance, _ := cbt.getBalance(chainID, id)
	go cbt.publishChangeEvent(id, before, currentBalance.GetBalance())
	return currentBalance.GetBalance()
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.AppEventIdentityRegistration) {
	switch event.Status {
	case registry.InProgress:
		cbt.alignWithTransactor(event.ChainID, event.ID)
	case registry.Registered:
		cbt.removeTransactorBounty(event.ChainID, event.ID)
		cbt.ForceBalanceUpdate(event.ChainID, event.ID)
	}
}

func (cbt *ConsumerBalanceTracker) alignWithTransactor(chainID int64, id identity.Identity) {
	balance, ok := cbt.getBalance(chainID, id)
	if ok {
		// do not override existing values with transactor data if it is not 0
		if balance.BCBalance.Cmp(big.NewInt(0)) != 0 {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	go func() {
		select {
		case <-cbt.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	bountyAmount := cbt.getTransactorBalance(ctx, chainID, id)
	if bountyAmount == nil {
		return
	}

	c := ConsumerBalance{
		BCBalance:          bountyAmount,
		BCSettled:          new(big.Int),
		GrandTotalPromised: new(big.Int),
	}

	cbt.setBalance(chainID, id, c)
	cbt.setTransactorBounty(chainID, id, bountyAmount)
	go cbt.publishChangeEvent(id, balance.GetBalance(), c.GetBalance())
}

func (cbt *ConsumerBalanceTracker) getTransactorBalance(ctx context.Context, chainID int64, id identity.Identity) *big.Int {
	data, err := cbt.identityRegistrationStatus(ctx, id, chainID)
	if err != nil {
		log.Error().Err(fmt.Errorf("could not fetch registration status from transactor: %w", err))
		return nil
	}

	if data.Status != registry.TransactorRegistrationEntryStatusCreated &&
		data.Status != registry.TransactorRegistrationEntryStatusPriceIncreased {
		return nil
	}

	if data.BountyAmount == nil || data.BountyAmount.Cmp(big.NewInt(0)) == 0 {
		// if we've got no bounty, get myst balance from BC and use that as bounty
		b, err := cbt.getUnregisteredChannelBalance(chainID, id)
		if err != nil {
			log.Error().Err(err).Msg("could not get unregistered balance")
			return nil
		}

		data.BountyAmount = b
	}

	log.Debug().Msgf("Loaded transactor state, current balance: %v MYST", data.BountyAmount)
	return data.BountyAmount
}

func (cbt *ConsumerBalanceTracker) recoverGrandTotalPromised(chainID int64, identity identity.Identity) error {
	var boff backoff.BackOff
	eback := backoff.NewExponentialBackOff()
	eback.MaxElapsedTime = time.Second * 20
	eback.InitialInterval = time.Second * 2

	boff = backoff.WithMaxRetries(eback, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	go func() {
		select {
		case <-cbt.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	var data HermesUserInfo
	boff = backoff.WithContext(boff, ctx)
	toRetry := func() error {
		d, err := cbt.consumerInfoGetter.GetConsumerData(chainID, identity.Address, time.Minute)
		if err != nil {
			if !errors.Is(err, ErrHermesNotFound) {
				return err
			}
			log.Debug().Msgf("No previous invoice grand total, assuming zero")
			return nil
		}
		data = d
		return nil
	}

	if err := backoff.Retry(toRetry, boff); err != nil {
		return err
	}

	latestPromised := big.NewInt(0)
	if data.LatestPromise.Amount != nil {
		latestPromised = data.LatestPromise.Amount
	}

	if isSettledBiggerThanPromised(data.Settled, latestPromised) {
		var err error
		latestPromised, err = cbt.getPromisedWhenSettledIsBigger(data, latestPromised, chainID, identity.ToCommonAddress())
		if err != nil {
			return err
		}
	}

	log.Debug().Msgf("Loaded hermes state: already promised: %v", latestPromised)

	hermes, err := cbt.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not get hermes address")
		return err
	}

	return cbt.consumerGrandTotalsStorage.Store(chainID, identity, hermes, latestPromised)
}

func (cbt *ConsumerBalanceTracker) handleStopEvent() {
	cbt.once.Do(func() {
		close(cbt.stop)
	})
}

func (cbt *ConsumerBalanceTracker) getBalance(chainID int64, id identity.Identity) (ConsumerBalance, bool) {
	cbt.balances.Lock()
	defer cbt.balances.Unlock()

	if v, ok := cbt.balances.valuesMap[newBalanceKey(chainID, id)]; ok {
		return v, true
	}

	return ConsumerBalance{
		BCBalance:          new(big.Int),
		BCSettled:          new(big.Int),
		GrandTotalPromised: new(big.Int),
	}, false
}

func (cbt *ConsumerBalanceTracker) setBalance(chainID int64, id identity.Identity, balance ConsumerBalance) {
	cbt.balances.Lock()
	defer cbt.balances.Unlock()

	cbt.balances.valuesMap[newBalanceKey(chainID, id)] = balance
}

func (cbt *ConsumerBalanceTracker) getTransactorBounty(chainID int64, id identity.Identity) (*big.Int, bool) {
	cbt.transactorBounties.Lock()
	defer cbt.transactorBounties.Unlock()

	if v, ok := cbt.transactorBounties.valuesMap[newBalanceKey(chainID, id)]; ok {
		return v, true
	}

	return nil, false
}

func (cbt *ConsumerBalanceTracker) setTransactorBounty(chainID int64, id identity.Identity, bountyAmount *big.Int) {
	cbt.transactorBounties.Lock()
	defer cbt.transactorBounties.Unlock()

	cbt.transactorBounties.valuesMap[newBalanceKey(chainID, id)] = bountyAmount
}

func (cbt *ConsumerBalanceTracker) removeTransactorBounty(chainID int64, id identity.Identity) {
	cbt.transactorBounties.Lock()
	defer cbt.transactorBounties.Unlock()

	delete(cbt.transactorBounties.valuesMap, newBalanceKey(chainID, id))
}

func (cbt *ConsumerBalanceTracker) updateGrandTotal(chainID int64, id identity.Identity, current *big.Int) {
	b, ok := cbt.getBalance(chainID, id)
	if !ok || b.OffchainNeedsSync() {
		cbt.ForceBalanceUpdate(chainID, id)
		return
	}

	before := b.BCBalance
	b.GrandTotalPromised = current
	cbt.setBalance(chainID, id, b)

	after, _ := cbt.getBalance(chainID, id)
	go cbt.publishChangeEvent(id, before, after.GetBalance())
}

// identityRegistrationStatus returns the registration status of a given identity.
func (cbt *ConsumerBalanceTracker) identityRegistrationStatus(ctx context.Context, id identity.Identity, chainID int64) (registry.TransactorStatusResponse, error) {
	var data registry.TransactorStatusResponse
	boff := backoff.WithContext(backoff.NewConstantBackOff(time.Millisecond*500), ctx)
	toRetry := func() error {
		resp, err := cbt.transactorRegistrationStatusProvider.FetchRegistrationStatus(id.Address)
		if err != nil {
			return err
		}

		var status *registry.TransactorStatusResponse
		for _, v := range resp {
			if v.ChainID == chainID {
				status = &v
				break
			}
		}

		if status == nil {
			err := fmt.Errorf("got response but failed to find status for id '%s' on chain '%d'", id.Address, chainID)
			return backoff.Permanent(err)
		}

		data = *status
		return nil
	}

	return data, backoff.Retry(toRetry, boff)
}

func (cbt *ConsumerBalanceTracker) getPromisedWhenSettledIsBigger(data HermesUserInfo, latestPromised *big.Int, chainID int64, identityAddress common.Address) (*big.Int, error) {
	if data.IsOffchain {
		return data.Settled, nil
	}

	activeChannelAddress, err := cbt.addressProvider.GetActiveChannelAddress(chainID, identityAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting active channel address: %w", err)
	}

	consumerHermes, err := cbt.blockchainInfoProvider.GetConsumerChannelsHermes(chainID, activeChannelAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting consumer channels hermes: %w", err)
	}

	return consumerHermes.Settled, nil
}

func safeSub(a, b *big.Int) *big.Int {
	if a == nil || b == nil {
		return new(big.Int)
	}

	if a.Cmp(b) >= 0 {
		return new(big.Int).Sub(a, b)
	}
	return new(big.Int)
}

func isSettledBiggerThanPromised(settled, promised *big.Int) bool {
	return settled != nil && settled.Cmp(promised) == 1
}

// ConsumerBalance represents the consumer balance
type ConsumerBalance struct {
	BCBalance          *big.Int
	BCSettled          *big.Int
	GrandTotalPromised *big.Int

	// IsOffchain is an optional indicator which marks an offchain balanace.
	// Offchain balances receive no updates on the blockchain and their
	// actual remaining balance should be retreived from hermes.
	IsOffchain       bool
	LastOffchainSync time.Time
}

// OffchainNeedsSync returns true if balance is offchain and should be synced.
func (cb ConsumerBalance) OffchainNeedsSync() bool {
	if !cb.IsOffchain {
		return false
	}

	if cb.LastOffchainSync.IsZero() {
		return true
	}

	expiresAfter := config.GetDuration(config.FlagOffchainBalanceExpiration)
	now := time.Now().UTC()
	return cb.LastOffchainSync.Add(expiresAfter).Before(now)
}

// GetBalance returns the current balance
func (cb ConsumerBalance) GetBalance() *big.Int {
	// Balance (to spend) = BCBalance - (hermesPromised - BCSettled)
	return safeSub(cb.BCBalance, safeSub(cb.GrandTotalPromised, cb.BCSettled))
}
