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
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/rs/zerolog/log"
)

// ConsumerBalanceTracker keeps track of consumer balances
type ConsumerBalanceTracker struct {
	balancesLock sync.Mutex
	balances     map[identity.Identity]Balance

	mystSCAddress            common.Address
	consumerBalanceChecker   consumerBalanceChecker
	channelAddressCalculator channelAddressCalculator
	publisher                eventbus.Publisher
}

// NewConsumerBalanceTracker creates a new instance
func NewConsumerBalanceTracker(publisher eventbus.Publisher, mystSCAddress common.Address, consumerBalanceChecker consumerBalanceChecker, channelAddressCalculator channelAddressCalculator) *ConsumerBalanceTracker {
	return &ConsumerBalanceTracker{
		balances:                 make(map[identity.Identity]Balance),
		consumerBalanceChecker:   consumerBalanceChecker,
		mystSCAddress:            mystSCAddress,
		publisher:                publisher,
		channelAddressCalculator: channelAddressCalculator,
	}
}

type consumerBalanceChecker interface {
	GetConsumerBalance(channel, mystSCAddress common.Address) (*big.Int, error)
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
	res, err := cbt.getBCBalance(identity)
	if err != nil {
		log.Error().Err(err).Msg("Could not get BC balance")
		return
	}
	cbt.updateBCBalance(identity, res)
}

func (cbt *ConsumerBalanceTracker) handleRegistrationEvent(event registry.RegistrationEventPayload) {
	switch event.Status {
	case registry.RegisteredConsumer, registry.RegisteredProvider:
		res, err := cbt.getBCBalance(event.ID)
		if err != nil {
			log.Error().Err(err).Msg("Could not get BC balance")
			return
		}
		cbt.updateBCBalance(event.ID, res)
	}
}

func (cbt *ConsumerBalanceTracker) decreaseBalance(id identity.Identity, b uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[id]; ok {
		if v.BCBalance != 0 {
			after := v.BCBalance - b
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

func (cbt *ConsumerBalanceTracker) updateBCBalance(id identity.Identity, b uint64) {
	cbt.balancesLock.Lock()
	defer cbt.balancesLock.Unlock()
	if v, ok := cbt.balances[id]; ok {
		v.BCBalance = b
		if v.CurrentEstimate == 0 {
			v.CurrentEstimate = b
		}
		cbt.balances[id] = v
	} else {
		cbt.balances[id] = Balance{
			BCBalance:       b,
			CurrentEstimate: b,
		}
	}
}

func (cbt *ConsumerBalanceTracker) getBCBalance(id identity.Identity) (uint64, error) {
	addr, err := cbt.channelAddressCalculator.GetChannelAddress(id)
	if err != nil {
		return 0, err
	}
	res, err := cbt.consumerBalanceChecker.GetConsumerBalance(addr, cbt.mystSCAddress)
	if err != nil {
		return 0, err
	}
	return res.Uint64(), nil
}
