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

// ConsumerBalanceTracker keeps track of consumer balances.
// TODO: this needs to take into account the saved state.
type ConsumerBalanceTracker struct {
	balancesLock sync.Mutex
	balances     map[identity.Identity]ConsumerBalance

	accountantAddress                    common.Address
	mystSCAddress                        common.Address
	consumerBalanceChecker               consumerBalanceChecker
	channelAddressCalculator             channelAddressCalculator
	publisher                            eventbus.Publisher
	consumerGrandTotalsStorage           consumerTotalsStorage
	consumerInfoGetter                   consumerInfoGetter
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider
	stop                                 chan struct{}
	once                                 sync.Once
}

type transactorRegistrationStatusProvider interface {
	FetchRegistrationStatus(id string) (registry.TransactorStatusResponse, error)
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(
	publisher eventbus.Publisher,
	mystSCAddress common.Address,
	accountantAddress common.Address,
	consumerBalanceChecker consumerBalanceChecker,
	channelAddressCalculator channelAddressCalculator,
	consumerGrandTotalsStorage consumerTotalsStorage,
	consumerInfoGetter consumerInfoGetter,
	transactorRegistrationStatusProvider transactorRegistrationStatusProvider,
) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                             make(map[identity.Identity]ConsumerBalance),
		consumerBalanceChecker:               consumerBalanceChecker,
		mystSCAddress:                        mystSCAddress,
		accountantAddress:                    accountantAddress,
		publisher:                            publisher,
		channelAddressCalculator:             channelAddressCalculator,
		consumerGrandTotalsStorage:           consumerGrandTotalsStorage,
		consumerInfoGetter:                   consumerInfoGetter,
		transactorRegistrationStatusProvider: transactorRegistrationStatusProvider,

		stop: make(chan struct{}),
	}
}

type consumerInfoGetter interface {
	GetConsumerData(id string) (ConsumerData, error)
}

type consumerBalanceChecker interface {
	SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error)
	GetConsumerChannel(addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error)
	GetMystBalance(mystAddress, identity common.Address) (*big.Int, error)
}

// Subscribe subscribes the consumer balance tracker to relevant events
func (cbt *ConsumerBalanceTracker) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(registry.AppTopicIdentityRegistration, cbt.handleRegistrationEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(registry.AppTopicTransactorTopUp, cbt.handleTopUpEvent)
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
func (cbt *ConsumerBalanceTracker) GetBalance(id identity.Identity) uint64 {
	if v, ok := cbt.getBalance(id); ok {
		return v.GetBalance()
	}
	return 0
}

func (cbt *ConsumerBalanceTracker) publishChangeEvent(id identity.Identity, before, after uint64) {
	if before == after {
		return
	}

	cbt.publisher.Publish(event.AppTopicBalanceChanged, event.AppEventBalanceChanged{
		Identity: id,
		Previous: before,
		Current:  after,
	})
}

func (cbt *ConsumerBalanceTracker) handleUnlockEvent(id string) {
	identity := identity.FromAddress(id)
	err := cbt.recoverGrandTotalPromised(identity)
	if err != nil {
		log.Error().Err(err).Msg("Could not recover Grand Total Promised")
	}
	cbt.ForceBalanceUpdate(identity)
}

func (cbt *ConsumerBalanceTracker) handleGrandTotalChanged(ev event.AppEventGrandTotalChanged) {
	if _, ok := cbt.getBalance(ev.ConsumerID); !ok {
		cbt.ForceBalanceUpdate(ev.ConsumerID)
		return
	}

	cbt.updateGrandTotal(ev.ConsumerID, ev.Current)
}

func (cbt *ConsumerBalanceTracker) handleTopUpEvent(id string) {
	addr, err := cbt.channelAddressCalculator.GetChannelAddress(identity.FromAddress(id))
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate channel address")
		return
	}
	sub, cancel, err := cbt.consumerBalanceChecker.SubscribeToConsumerBalanceEvent(addr, cbt.mystSCAddress, time.Minute*15)
	if err != nil {
		log.Error().Err(err).Msg("Could not subscribe to consumer balance event")
		return
	}

	updated := false
	defer cancel()
	select {
	case ev, more := <-sub:
		if !more {
			// in case of a timeout, force update
			if !updated {
				cbt.ForceBalanceUpdate(identity.FromAddress(id))
			}
			return
		}
		updated = true
		cbt.increaseBCBalance(identity.FromAddress(id), ev.Value.Uint64())
	case <-cbt.stop:
		return
	}
}

// ForceBalanceUpdate forces a balance update and returns the updated balance
func (cbt *ConsumerBalanceTracker) ForceBalanceUpdate(id identity.Identity) uint64 {
	fallback := cbt.GetBalance(id)

	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate channel address")
		return fallback
	}

	cc, err := cbt.consumerBalanceChecker.GetConsumerChannel(addr, cbt.mystSCAddress)
	if err != nil {
		log.Error().Err(err).Msg("Could not get consumer channel")
		return fallback
	}

	grandTotal, err := cbt.consumerGrandTotalsStorage.Get(id, cbt.accountantAddress)
	if errors.Is(err, ErrNotFound) {
		if err := cbt.recoverGrandTotalPromised(id); err != nil {
			log.Error().Err(err).Msg("Could not recover Grand Total Promised")
		}
		grandTotal, err = cbt.consumerGrandTotalsStorage.Get(id, cbt.accountantAddress)
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Error().Err(err).Msg("Could not get consumer grand total promised")
		return fallback
	}

	var before uint64
	if v, ok := cbt.getBalance(id); ok {
		before = v.GetBalance()
	}

	cbt.setBalance(id, ConsumerBalance{
		BCBalance:          cc.Balance.Uint64(),
		BCSettled:          cc.Settled.Uint64(),
		GrandTotalPromised: grandTotal,
	})

	currentBalance, _ := cbt.getBalance(id)
	go cbt.publishChangeEvent(id, before, currentBalance.GetBalance())
	return currentBalance.GetBalance()
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.AppEventIdentityRegistration) {
	switch event.Status {
	case registry.InProgress:
		cbt.alignWithTransactor(event.ID)
	case registry.Registered:
		cbt.ForceBalanceUpdate(event.ID)
	}
}

func (cbt *ConsumerBalanceTracker) alignWithTransactor(id identity.Identity) {
	balance, ok := cbt.getBalance(id)
	if ok {
		// do not override existing values with transactor data
		return
	}

	// do not override existing balances with transactor data
	if balance.BCBalance != 0 {
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
		data = resp
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

	if data.BountyAmount == 0 {
		// if we've got no bounty, get myst balance from BC and use that as bounty
		addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
		if err != nil {
			log.Error().Err(err).Msg("could not compute channel address")
			return
		}
		balance, err := cbt.consumerBalanceChecker.GetMystBalance(cbt.mystSCAddress, addr)
		if err != nil {
			log.Error().Err(err).Msg("could not get myst balance on consumer channel")
			return
		}

		data.BountyAmount = balance.Uint64()
	}

	c := ConsumerBalance{
		BCBalance:          data.BountyAmount,
		BCSettled:          0,
		GrandTotalPromised: 0,
	}
	log.Debug().Msgf("Loaded transactor state, current balance: %v MYST", data.BountyAmount)
	cbt.setBalance(id, c)
	go cbt.publishChangeEvent(id, balance.GetBalance(), c.GetBalance())
}

func (cbt *ConsumerBalanceTracker) recoverGrandTotalPromised(identity identity.Identity) error {
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
		d, err := cbt.consumerInfoGetter.GetConsumerData(identity.Address)
		if err != nil {
			if err != ErrAccountantNotFound {
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

	log.Debug().Msgf("Loaded accountant state: already promised: %v", data.LatestPromise.Amount)
	return cbt.consumerGrandTotalsStorage.Store(identity, cbt.accountantAddress, data.LatestPromise.Amount)
}

func (cbt *ConsumerBalanceTracker) handleStopEvent() {
	cbt.once.Do(func() {
		close(cbt.stop)
	})
}

func (cbt *ConsumerBalanceTracker) increaseBCBalance(id identity.Identity, diff uint64) {
	b, ok := cbt.getBalance(id)
	before := b.BCBalance
	if ok {
		b.BCBalance = safeAdd(b.BCBalance, diff)
		cbt.setBalance(id, b)
	} else {
		cbt.ForceBalanceUpdate(id)
	}
	after, _ := cbt.getBalance(id)

	go cbt.publishChangeEvent(id, before, after.GetBalance())
}

func (cbt *ConsumerBalanceTracker) getBalance(id identity.Identity) (ConsumerBalance, bool) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	if v, ok := cbt.balances[id]; ok {
		return v, true
	}

	return ConsumerBalance{}, false
}

func (cbt *ConsumerBalanceTracker) setBalance(id identity.Identity, balance ConsumerBalance) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	cbt.balances[id] = balance
}

func (cbt *ConsumerBalanceTracker) updateGrandTotal(id identity.Identity, current uint64) {
	b, ok := cbt.getBalance(id)
	before := b.BCBalance
	if ok {
		b.GrandTotalPromised = current
		cbt.setBalance(id, b)
	} else {
		cbt.ForceBalanceUpdate(id)
	}
	after, _ := cbt.getBalance(id)

	go cbt.publishChangeEvent(id, before, after.GetBalance())
}

func safeSub(a, b uint64) uint64 {
	if a >= b {
		return a - b
	}
	return 0
}

func safeAdd(a, b uint64) uint64 {
	c := a + b
	if (c > a) == (b > 0) {
		return c
	}
	return 0
}

// ConsumerBalance represents the consumer balance
type ConsumerBalance struct {
	BCBalance          uint64
	BCSettled          uint64
	GrandTotalPromised uint64
}

// GetBalance returns the current balance
func (cb ConsumerBalance) GetBalance() uint64 {
	// Balance (to spend) = BCBalance - (accountantPromised - BCSettled)
	return safeSub(cb.BCBalance, safeSub(cb.GrandTotalPromised, cb.BCSettled))
}
