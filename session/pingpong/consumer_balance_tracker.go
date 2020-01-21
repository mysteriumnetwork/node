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

	"github.com/ethereum/go-ethereum/common"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/rs/zerolog/log"
)

type accountantBalanceFetcher func(id string) (ConsumerData, error)

// ConsumerBalanceTracker keeps track of consumer balances.
// TODO: this needs to take into account the saved state.
type ConsumerBalanceTracker struct {
	balancesLock sync.Mutex
	balances     map[identity.Identity]Balance

	mystSCAddress            common.Address
	consumerBalanceChecker   consumerBalanceChecker
	channelAddressCalculator channelAddressCalculator
	accountantBalanceFetcher accountantBalanceFetcher
	publisher                eventbus.Publisher

	stop chan struct{}
	once sync.Once
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(publisher eventbus.Publisher, mystSCAddress common.Address, consumerBalanceChecker consumerBalanceChecker, channelAddressCalculator channelAddressCalculator, accountantBalanceFetcher accountantBalanceFetcher) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                 make(map[identity.Identity]Balance),
		consumerBalanceChecker:   consumerBalanceChecker,
		mystSCAddress:            mystSCAddress,
		publisher:                publisher,
		channelAddressCalculator: channelAddressCalculator,
		accountantBalanceFetcher: accountantBalanceFetcher,
		stop:                     make(chan struct{}),
	}
}

type consumerBalanceChecker interface {
	SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address) (chan *bindings.MystTokenTransfer, func(), error)
}

// Balance represents the balance
type Balance struct {
	BCBalance       uint64
	CurrentEstimate uint64
}

// Subscribe subscribes the consumer balance tracker to relevant events
func (cbt *ConsumerBalanceTracker) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(registry.RegistrationEventTopic, cbt.handleRegistrationEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(registry.TransactorTopUpTopic, cbt.handleTopUpEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(string(nodevent.StatusStopped), cbt.handleStopEvent)
	if err != nil {
		return err
	}
	err = bus.SubscribeAsync(ExchangeMessageTopic, cbt.handleExchangeMessageEvent)
	if err != nil {
		return err
	}
	return bus.SubscribeAsync(identity.IdentityUnlockTopic, cbt.handleUnlockEvent)
}

// GetBalance gets the current balance for given identity
func (cbt *ConsumerBalanceTracker) GetBalance(ID identity.Identity) uint64 {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[ID]; ok {
		return v.CurrentEstimate
	}
	return 0
}

func (cbt *ConsumerBalanceTracker) handleExchangeMessageEvent(event ExchangeMessageEventPayload) {
	cbt.decreaseBalance(event.Identity, event.AmountPromised)
}

func (cbt *ConsumerBalanceTracker) publishChangeEvent(id identity.Identity, before, after uint64) {
	cbt.publisher.Publish(BalanceChangedTopic, BalanceChangedEvent{
		Identity: id,
		Previous: before,
		Current:  after,
	})
}

func (cbt *ConsumerBalanceTracker) handleUnlockEvent(id string) {
	identity := identity.FromAddress(id)
	cbt.updateBalanceFromAccountant(identity)
}

func (cbt *ConsumerBalanceTracker) handleTopUpEvent(id string) {
	addr, err := cbt.channelAddressCalculator.GetChannelAddress(identity.FromAddress(id))
	if err != nil {
		log.Error().Err(err).Msg("Could not generate channel address")
		return
	}
	sub, cancel, err := cbt.consumerBalanceChecker.SubscribeToConsumerBalanceEvent(addr, cbt.mystSCAddress)
	if err != nil {
		log.Error().Err(err).Msg("Could not subscribe to consumer balance event")
		return
	}
	defer cancel()
	select {
	case ev, more := <-sub:
		if !more {
			return
		}
		cbt.increaseBalance(identity.FromAddress(id), ev.Value.Uint64())
	case <-cbt.stop:
		return
	}
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.RegistrationEventPayload) {
	switch event.Status {
	case registry.RegisteredConsumer, registry.RegisteredProvider:
		cbt.updateBalanceFromAccountant(event.ID)
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
	if v, ok := cbt.balances[id]; ok {
		v.CurrentEstimate += b
		v.BCBalance += b
		cbt.balances[id] = v
		go cbt.publishChangeEvent(id, v.CurrentEstimate, v.BCBalance)
	} else {
		cbt.balances[id] = Balance{
			BCBalance:       b,
			CurrentEstimate: b,
		}
		go cbt.publishChangeEvent(id, 0, b)
	}
}

func (cbt *ConsumerBalanceTracker) decreaseBalance(id identity.Identity, b uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[id]; ok {
		if v.BCBalance != 0 {
			after := safeSub(v.BCBalance, b)
			go cbt.publishChangeEvent(id, v.CurrentEstimate, after)
			v.CurrentEstimate = after
			cbt.balances[id] = v
		}
	} else {
		cbt.balances[id] = Balance{
			BCBalance:       0,
			CurrentEstimate: 0,
		}
		go cbt.publishChangeEvent(id, 0, 0)
	}
}

func (cbt *ConsumerBalanceTracker) updateBalanceFromAccountant(id identity.Identity) {
	cb, err := cbt.accountantBalanceFetcher(id.Address)
	if err != nil {
		log.Error().Err(err).Msg("could not get accountant balance")
		return
	}

	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[id]; ok {
		isIncreased := true
		var diff uint64

		if v.BCBalance >= cb.Balance {
			isIncreased = false
			diff = safeSub(v.BCBalance, cb.Balance)
		} else {
			diff = safeSub(cb.Balance, v.BCBalance)
		}

		before := cbt.balances[id].CurrentEstimate
		if isIncreased {
			cbt.balances[id] = Balance{
				BCBalance:       v.BCBalance + diff,
				CurrentEstimate: v.CurrentEstimate + diff,
			}
		} else {
			cbt.balances[id] = Balance{
				BCBalance:       safeSub(v.BCBalance, diff),
				CurrentEstimate: safeSub(v.CurrentEstimate, diff),
			}
		}
		go cbt.publishChangeEvent(id, before, cbt.balances[id].CurrentEstimate)
	} else {
		cbt.balances[id] = Balance{
			BCBalance:       cb.Balance,
			CurrentEstimate: cb.Balance,
		}
		go cbt.publishChangeEvent(id, 0, cb.Balance)
	}
}

func safeSub(a, b uint64) uint64 {
	if a >= b {
		return a - b
	}
	return 0
}
