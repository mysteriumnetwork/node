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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/rs/zerolog/log"
)

// ConsumerBalanceTracker keeps track of consumer balances.
// TODO: this needs to take into account the saved state.
type ConsumerBalanceTracker struct {
	balancesLock sync.Mutex
	balances     map[identity.Identity]uint64

	accountantAddress        common.Address
	mystSCAddress            common.Address
	consumerBalanceChecker   consumerBalanceChecker
	channelAddressCalculator channelAddressCalculator
	publisher                eventbus.Publisher
	consumerGrandTotalGetter consumerGrandTotalGetter

	stop chan struct{}
	once sync.Once
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(
	publisher eventbus.Publisher,
	mystSCAddress, accountantAddress common.Address,
	consumerBalanceChecker consumerBalanceChecker,
	channelAddressCalculator channelAddressCalculator,
	consumerGrandTotalGetter consumerGrandTotalGetter,
) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                 make(map[identity.Identity]uint64),
		consumerBalanceChecker:   consumerBalanceChecker,
		mystSCAddress:            mystSCAddress,
		accountantAddress:        accountantAddress,
		publisher:                publisher,
		channelAddressCalculator: channelAddressCalculator,
		consumerGrandTotalGetter: consumerGrandTotalGetter,
		stop:                     make(chan struct{}),
	}
}

type consumerGrandTotalGetter interface {
	Get(consumerAddress, accountantAddress string) (uint64, error)
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
	err = bus.SubscribeAsync(AppTopicExchangeMessage, cbt.handleExchangeMessageEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(AppTopicGrandTotalRecovered, cbt.handleGrandTotalRecovered)
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
		return v
	}
	return 0
}

func (cbt *ConsumerBalanceTracker) handleExchangeMessageEvent(event AppEventExchangeMessage) {
	cbt.decreaseBalance(event.Identity, event.AmountPromised)
}

func (cbt *ConsumerBalanceTracker) publishChangeEvent(id identity.Identity, before, after uint64) {
	cbt.publisher.Publish(AppTopicBalanceChanged, AppEventBalanceChanged{
		Identity: id,
		Previous: before,
		Current:  after,
	})
}

func (cbt *ConsumerBalanceTracker) handleUnlockEvent(id string) {
	identity := identity.FromAddress(id)
	cbt.ForceBalanceUpdate(identity)
}

func (cbt *ConsumerBalanceTracker) handleGrandTotalRecovered(ev GrandTotalRecovered) {
	cbt.ForceBalanceUpdate(ev.Identity)
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
		cbt.increaseBalance(identity.FromAddress(id), ev.Value.Uint64())
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

	grandTotal, err := cbt.consumerGrandTotalGetter.Get(id.Address, identity.FromAddress(cbt.accountantAddress.Hex()).Address)
	if err != nil && err != ErrNotFound {
		log.Error().Err(err).Msg("Could not get consumer grand total promised")
		return 0
	}

	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	// Balance (to spend) = BCBalance - (accountantPromised - BCSettled)
	diff := safeSub(grandTotal, cc.Settled.Uint64())

	currentBalance := safeSub(cc.Balance.Uint64(), diff)

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v
	}

	cbt.balances[id] = currentBalance
	go cbt.publishChangeEvent(id, before, currentBalance)
	return currentBalance
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.AppEventIdentityRegistration) {
	switch event.Status {
	case registry.RegisteredConsumer, registry.RegisteredProvider:
		cbt.ForceBalanceUpdate(event.ID)
	}
}

func (cbt *ConsumerBalanceTracker) handleStopEvent() {
	cbt.once.Do(func() {
		close(cbt.stop)
	})
}

func (cbt *ConsumerBalanceTracker) increaseBalance(id identity.Identity, b uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v
	}

	current := safeAdd(before, b)
	cbt.balances[id] = current

	go cbt.publishChangeEvent(id, before, current)
}

func (cbt *ConsumerBalanceTracker) decreaseBalance(id identity.Identity, b uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()

	var before uint64
	if v, ok := cbt.balances[id]; ok {
		before = v
	}

	current := safeSub(before, b)
	cbt.balances[id] = current

	go cbt.publishChangeEvent(id, before, current)
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
