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

package balance

import (
	"time"

	"github.com/mysteriumnetwork/node/money"
)

const balanceTrackerPrefix = "[balance-tracker] "

// PeerSender knows how to send a balance message to the peer
type PeerSender interface {
	Send(Message) error
}

// TimeKeeper keeps track of time for payments
type TimeKeeper interface {
	StartTracking()
	Elapsed() time.Duration
}

// AmountCalculator is able to deduce the amount required for payment from a given duraiton
type AmountCalculator interface {
	TotalAmount(duration time.Duration) money.Money
}

// ProviderBalanceTracker is responsible for tracking the balance on the provider side
type ProviderBalanceTracker struct {
	sender           PeerSender
	timeKeeper       TimeKeeper
	amountCalculator AmountCalculator
	period           time.Duration

	totalPromised uint64
	balance       uint64
	stop          chan struct{}
}

// NewProviderBalanceTracker returns a new instance of the providerBalanceTracker
func NewProviderBalanceTracker(timeKeeper TimeKeeper, amountCalculator AmountCalculator, period time.Duration, initialBalance uint64) *ProviderBalanceTracker {
	return &ProviderBalanceTracker{
		timeKeeper:       timeKeeper,
		period:           period,
		amountCalculator: amountCalculator,
		totalPromised:    initialBalance,

		stop: make(chan struct{}),
	}
}

func (pbt *ProviderBalanceTracker) calculateBalance() {
	cost := pbt.amountCalculator.TotalAmount(pbt.timeKeeper.Elapsed())
	pbt.balance = pbt.totalPromised - cost.Amount
}

// GetBalance returns the balance message
func (pbt *ProviderBalanceTracker) GetBalance() Message {
	pbt.calculateBalance()
	// TODO: sequence ID should come here, somehow
	return Message{
		SequenceID: 0,
		Balance:    pbt.balance,
	}
}
