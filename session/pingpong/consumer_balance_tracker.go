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
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/rs/zerolog/log"
)

type balanceKey string

func newBalanceKey(chainID int64, id identity.Identity) balanceKey {
	return balanceKey(fmt.Sprintf("%v_%v", id.Address, chainID))
}

// ConsumerBalanceTracker keeps track of consumer balances.
// TODO: this needs to take into account the saved state.
type ConsumerBalanceTracker struct {
	balancesLock sync.Mutex
	balances     map[balanceKey]ConsumerBalance

	registry                             registrationStatusProvider
	hermesAddress                        common.Address
	mystSCAddress                        common.Address
	consumerBalanceChecker               consumerBalanceChecker
	channelAddressCalculator             channelAddressCalculator
	bus                                  eventbus.EventBus
	consumerGrandTotalsStorage           consumerTotalsStorage
	consumerInfoGetter                   consumerInfoGetter
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider
	stop                                 chan struct{}
	once                                 sync.Once
}

type transactorRegistrationStatusProvider interface {
	FetchRegistrationStatus(id string) ([]registry.TransactorStatusResponse, error)
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(
	publisher eventbus.EventBus,
	mystSCAddress common.Address,
	hermesAddress common.Address,
	consumerBalanceChecker consumerBalanceChecker,
	channelAddressCalculator channelAddressCalculator,
	consumerGrandTotalsStorage consumerTotalsStorage,
	consumerInfoGetter consumerInfoGetter,
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider,
	registry registrationStatusProvider,
) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                             make(map[balanceKey]ConsumerBalance),
		consumerBalanceChecker:               consumerBalanceChecker,
		mystSCAddress:                        mystSCAddress,
		hermesAddress:                        hermesAddress,
		bus:                                  publisher,
		channelAddressCalculator:             channelAddressCalculator,
		consumerGrandTotalsStorage:           consumerGrandTotalsStorage,
		consumerInfoGetter:                   consumerInfoGetter,
		transactorRegistrationStatusProvider: transactorRegistrationStatusProvider,
		registry:                             registry,
		stop:                                 make(chan struct{}),
	}
}

type consumerInfoGetter interface {
	GetConsumerData(chainID int64, id string) (ConsumerData, error)
}

type consumerBalanceChecker interface {
	SubscribeToConsumerBalanceEvent(chainID int64, channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error)
	GetConsumerChannel(chainID int64, addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error)
	GetMystBalance(chainID int64, mystAddress, identity common.Address) (*big.Int, error)
}

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
	return bus.SubscribeAsync(identity.AppTopicIdentityUnlock, cbt.handleUnlockEvent)
}

// GetBalance gets the current balance for given identity
func (cbt *ConsumerBalanceTracker) GetBalance(chainID int64, id identity.Identity) *big.Int {
	if v, ok := cbt.getBalance(chainID, id); ok {
		return v.GetBalance()
	}
	return new(big.Int)
}

func (cbt *ConsumerBalanceTracker) publishChangeEvent(id identity.Identity, before, after *big.Int) {
	if before == after {
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

	go cbt.subscribeToExternalChannelTopup(data.ChainID, data.ID)
}

func (cbt *ConsumerBalanceTracker) handleGrandTotalChanged(ev event.AppEventGrandTotalChanged) {
	if _, ok := cbt.getBalance(ev.ChainID, ev.ConsumerID); !ok {
		cbt.ForceBalanceUpdate(ev.ChainID, ev.ConsumerID)
		return
	}

	cbt.updateGrandTotal(ev.ChainID, ev.ConsumerID, ev.Current)
}

func (cbt *ConsumerBalanceTracker) getUnregisteredChannelBalance(chainID int64, id identity.Identity) *big.Int {
	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		log.Error().Err(err).Msg("could not compute channel address")
		return new(big.Int)
	}
	balance, err := cbt.consumerBalanceChecker.GetMystBalance(chainID, cbt.mystSCAddress, addr)
	if err != nil {
		log.Error().Err(err).Msg("could not get myst balance on consumer channel")
		return new(big.Int)
	}
	return balance
}

func (cbt *ConsumerBalanceTracker) subscribeToExternalChannelTopup(chainID int64, id identity.Identity) {
	// if we've been stopped, don't re-start
	select {
	case <-cbt.stop:
		return
	default:
		break
	}

	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		log.Error().Err(err).Msg("could not compute channel address")
		return
	}

	ev, cancel, err := cbt.consumerBalanceChecker.SubscribeToConsumerBalanceEvent(chainID, addr, cbt.mystSCAddress, time.Hour*72)
	if err != nil {
		log.Error().Err(err).Msg("could not subscribe to channel balance events")
		return
	}
	defer cancel()
	log.Info().Msgf("Subscribed to channel %v balance events", addr.Hex())

	go func() {
		<-cbt.stop
		// cancel closes ev, so no need to close it.
		cancel()
	}()

	cbt.bus.Subscribe(registry.AppTopicEthereumClientReconnected, func(interface{}) {
		cancel()
	})

	func() {
		defer func() {
			// we've been interrupted, restart
			go cbt.subscribeToExternalChannelTopup(chainID, id)
		}()

		for e := range ev {
			if e == nil {
				return
			}

			previous, _ := cbt.getBalance(chainID, id)
			if bytes.Equal(e.To.Bytes(), addr.Bytes()) {
				cbt.setBalance(chainID, id, ConsumerBalance{
					BCBalance:          new(big.Int).Add(previous.BCBalance, e.Value),
					BCSettled:          previous.BCSettled,
					GrandTotalPromised: previous.GrandTotalPromised,
				})
			} else {
				cbt.setBalance(chainID, id, ConsumerBalance{
					BCBalance:          new(big.Int).Sub(previous.BCBalance, e.Value),
					BCSettled:          previous.BCSettled,
					GrandTotalPromised: previous.GrandTotalPromised,
				})
			}
			currentBalance, _ := cbt.getBalance(chainID, id)
			go cbt.publishChangeEvent(id, previous.GetBalance(), currentBalance.GetBalance())
		}
	}()
}

// ForceBalanceUpdate forces a balance update and returns the updated balance
func (cbt *ConsumerBalanceTracker) ForceBalanceUpdate(chainID int64, id identity.Identity) *big.Int {
	fallback := cbt.GetBalance(chainID, id)

	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate channel address")
		return fallback
	}

	cc, err := cbt.consumerBalanceChecker.GetConsumerChannel(chainID, addr, cbt.mystSCAddress)
	if err != nil {
		log.Error().Err(err).Msg("Could not get consumer channel")
		// This indicates we're not registered, check for unregistered balance.
		unregisteredBalance := cbt.getUnregisteredChannelBalance(chainID, id)
		// We'll also launch a goroutine to listen for external top up.
		cbt.setBalance(chainID, id, ConsumerBalance{
			BCBalance:          unregisteredBalance,
			BCSettled:          new(big.Int),
			GrandTotalPromised: new(big.Int),
		})

		currentBalance, _ := cbt.getBalance(chainID, id)
		go cbt.publishChangeEvent(id, new(big.Int), currentBalance.GetBalance())
		return unregisteredBalance
	}

	grandTotal, err := cbt.consumerGrandTotalsStorage.Get(chainID, id, cbt.hermesAddress)
	if errors.Is(err, ErrNotFound) {
		if err := cbt.recoverGrandTotalPromised(chainID, id); err != nil {
			log.Error().Err(err).Msg("Could not recover Grand Total Promised")
		}
		grandTotal, err = cbt.consumerGrandTotalsStorage.Get(chainID, id, cbt.hermesAddress)
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Error().Err(err).Msg("Could not get consumer grand total promised")
		return fallback
	}

	var before = new(big.Int)
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
		cbt.ForceBalanceUpdate(event.ChainID, event.ID)
	}
}

func (cbt *ConsumerBalanceTracker) alignWithTransactor(chainID int64, id identity.Identity) {
	balance, ok := cbt.getBalance(chainID, id)
	if ok {
		// do not override existing values with transactor data
		return
	}

	// do not override existing balances with transactor data
	if balance.BCBalance.Cmp(big.NewInt(0)) != 0 {
		return
	}

	var boff backoff.BackOff
	eback := backoff.NewConstantBackOff(time.Millisecond * 500)
	boff = backoff.WithMaxRetries(eback, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-cbt.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	var data registry.TransactorStatusResponse
	boff = backoff.WithContext(boff, ctx)
	toRetry := func() error {
		resp, err := cbt.transactorRegistrationStatusProvider.FetchRegistrationStatus(id.Address)
		if err != nil {
			return err
		}

		var status *registry.TransactorStatusResponse
		for _, v := range resp {
			if v.ChainID != chainID {
				continue
			}
			status = &v
			break
		}

		if status == nil {
			err := fmt.Errorf("got response but failed to find status for id '%s' on chain '%d'", id.Address, chainID)
			return backoff.Permanent(err)
		}

		data = *status
		return nil
	}

	if err := backoff.Retry(toRetry, boff); err != nil {
		log.Error().Err(fmt.Errorf("could not fetch registration status from transactor: %w", err))
		return
	}

	if data.Status != registry.TransactorRegistrationEntryStatusCreated &&
		data.Status != registry.TransactorRegistrationEntryStatusPriceIncreased {
		return
	}

	if data.BountyAmount.Cmp(big.NewInt(0)) == 0 {
		// if we've got no bounty, get myst balance from BC and use that as bounty
		b := cbt.getUnregisteredChannelBalance(chainID, id)
		data.BountyAmount = b
	}

	c := ConsumerBalance{
		BCBalance:          data.BountyAmount,
		BCSettled:          new(big.Int),
		GrandTotalPromised: new(big.Int),
	}
	log.Debug().Msgf("Loaded transactor state, current balance: %v MYST", data.BountyAmount)
	cbt.setBalance(chainID, id, c)
	go cbt.publishChangeEvent(id, balance.GetBalance(), c.GetBalance())
}

func (cbt *ConsumerBalanceTracker) recoverGrandTotalPromised(chainID int64, identity identity.Identity) error {
	var boff backoff.BackOff
	eback := backoff.NewExponentialBackOff()
	eback.MaxElapsedTime = time.Second * 20
	eback.InitialInterval = time.Second * 2

	boff = backoff.WithMaxRetries(eback, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-cbt.stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	var data ConsumerData
	boff = backoff.WithContext(boff, ctx)
	toRetry := func() error {
		d, err := cbt.consumerInfoGetter.GetConsumerData(chainID, identity.Address)
		if err != nil {
			if err != ErrHermesNotFound {
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

	if data.LatestPromise.Amount == nil {
		data.LatestPromise.Amount = new(big.Int)
	}

	log.Debug().Msgf("Loaded hermes state: already promised: %v", data.LatestPromise.Amount)
	return cbt.consumerGrandTotalsStorage.Store(chainID, identity, cbt.hermesAddress, data.LatestPromise.Amount)
}

func (cbt *ConsumerBalanceTracker) handleStopEvent() {
	cbt.once.Do(func() {
		close(cbt.stop)
	})
}

func (cbt *ConsumerBalanceTracker) increaseBCBalance(chainID int64, id identity.Identity, diff *big.Int) {
	b, ok := cbt.getBalance(chainID, id)
	before := b.BCBalance
	if ok {
		b.BCBalance = new(big.Int).Add(b.BCBalance, diff)
		cbt.setBalance(chainID, id, b)
	} else {
		cbt.ForceBalanceUpdate(chainID, id)
	}
	after, _ := cbt.getBalance(chainID, id)

	go cbt.publishChangeEvent(id, before, after.GetBalance())
}

func (cbt *ConsumerBalanceTracker) getBalance(chainID int64, id identity.Identity) (ConsumerBalance, bool) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	if v, ok := cbt.balances[newBalanceKey(chainID, id)]; ok {
		return v, true
	}

	return ConsumerBalance{
		BCBalance:          new(big.Int),
		BCSettled:          new(big.Int),
		GrandTotalPromised: new(big.Int),
	}, false
}

func (cbt *ConsumerBalanceTracker) setBalance(chainID int64, id identity.Identity, balance ConsumerBalance) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	cbt.balances[newBalanceKey(chainID, id)] = balance
}

func (cbt *ConsumerBalanceTracker) updateGrandTotal(chainID int64, id identity.Identity, current *big.Int) {
	b, ok := cbt.getBalance(chainID, id)
	before := b.BCBalance
	if ok {
		b.GrandTotalPromised = current
		cbt.setBalance(chainID, id, b)
	} else {
		cbt.ForceBalanceUpdate(chainID, id)
	}

	after, _ := cbt.getBalance(chainID, id)
	go cbt.publishChangeEvent(id, before, after.GetBalance())
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

// ConsumerBalance represents the consumer balance
type ConsumerBalance struct {
	BCBalance          *big.Int
	BCSettled          *big.Int
	GrandTotalPromised *big.Int
}

// GetBalance returns the current balance
func (cb ConsumerBalance) GetBalance() *big.Int {
	// Balance (to spend) = BCBalance - (hermesPromised - BCSettled)
	return safeSub(cb.BCBalance, safeSub(cb.GrandTotalPromised, cb.BCSettled))
}
