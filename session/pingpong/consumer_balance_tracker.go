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

	accountantAddress          identity.Identity
	mystSCAddress              common.Address
	consumerBalanceChecker     consumerBalanceChecker
	channelAddressCalculator   channelAddressCalculator
	publisher                  eventbus.Publisher
	consumerGrandTotalsStorage consumerTotalsStorage
	consumerInfoGetter         consumerInfoGetter

	stop chan struct{}
	once sync.Once
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(
	publisher eventbus.Publisher,
	mystSCAddress common.Address,
	accountantAddress identity.Identity,
	consumerBalanceChecker consumerBalanceChecker,
	channelAddressCalculator channelAddressCalculator,
	consumerGrandTotalsStorage consumerTotalsStorage,
	consumerInfoGetter consumerInfoGetter,
) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                   make(map[identity.Identity]ConsumerBalance),
		consumerBalanceChecker:     consumerBalanceChecker,
		mystSCAddress:              mystSCAddress,
		accountantAddress:          accountantAddress,
		publisher:                  publisher,
		channelAddressCalculator:   channelAddressCalculator,
		consumerGrandTotalsStorage: consumerGrandTotalsStorage,
		consumerInfoGetter:         consumerInfoGetter,

		stop: make(chan struct{}),
	}
}

type consumerInfoGetter interface {
	GetConsumerData(id string) (ConsumerData, error)
}

type consumerBalanceChecker interface {
	SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error)
	GetConsumerChannel(addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error)
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
func (cbt *ConsumerBalanceTracker) GetBalance(ID identity.Identity) uint64 {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[ID]; ok {
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
	cbt.balancesLock.Lock()
	_, ok := cbt.balances[ev.ConsumerID]
	cbt.balancesLock.Unlock()

	if !ok {
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
	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		log.Error().Err(err).Msg("Could not calculate channel address")
		return 0
	}

	cc, err := cbt.consumerBalanceChecker.GetConsumerChannel(addr, cbt.mystSCAddress)
	if err != nil {
		log.Error().Err(err).Msg("Could not get consumer channel")
		return 0
	}

	grandTotal, err := cbt.consumerGrandTotalsStorage.Get(id, cbt.accountantAddress)
	if err != nil && err != ErrNotFound {
		log.Error().Err(err).Msg("Could not get consumer grand total promised")
		return 0
	}

	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v.GetBalance()
	}

	cbt.balances[id] = ConsumerBalance{
		BCBalance:          cc.Balance.Uint64(),
		BCSettled:          cc.Settled.Uint64(),
		GrandTotalPromised: grandTotal,
	}

	currentBalance := cbt.balances[id].GetBalance()
	go cbt.publishChangeEvent(id, before, currentBalance)
	return currentBalance
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.AppEventIdentityRegistration) {
	switch event.Status {
	case registry.RegisteredConsumer, registry.RegisteredProvider:
		cbt.ForceBalanceUpdate(event.ID)
	}
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
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v.GetBalance()
		v.BCBalance = safeAdd(v.BCBalance, diff)
		cbt.balances[id] = v
	} else {
		cbt.ForceBalanceUpdate(id)
	}

	go cbt.publishChangeEvent(id, before, cbt.balances[id].GetBalance())
}

func (cbt *ConsumerBalanceTracker) updateGrandTotal(id identity.Identity, current uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v.GetBalance()
		v.GrandTotalPromised = current
		cbt.balances[id] = v
	} else {
		cbt.ForceBalanceUpdate(id)
	}

	go cbt.publishChangeEvent(id, before, cbt.balances[id].GetBalance())
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
