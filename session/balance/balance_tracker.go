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
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/money"
)

// TimeKeeper keeps track of time for payments
type TimeKeeper interface {
	StartTracking()
	Elapsed() time.Duration
}

// AmountCalculator is able to deduce the amount required for payment from a given duration
type AmountCalculator interface {
	TotalAmount(duration time.Duration) money.Money
}

// BalanceTracker is responsible for tracking the balance on the provider side
type BalanceTracker struct {
	timeKeeper       TimeKeeper
	amountCalculator AmountCalculator

	totalPromised uint64
	balance       uint64

	sync.Mutex
}

// NewBalanceTracker returns a new instance of the providerBalanceTracker
func NewBalanceTracker(timeKeeper TimeKeeper, amountCalculator AmountCalculator, initialBalance uint64) *BalanceTracker {
	return &BalanceTracker{
		timeKeeper:       timeKeeper,
		amountCalculator: amountCalculator,
		totalPromised:    initialBalance,
	}
}

func (bt *BalanceTracker) calculateBalance() {
	bt.Lock()
	defer bt.Unlock()
	cost := bt.amountCalculator.TotalAmount(bt.timeKeeper.Elapsed())
	bt.balance = bt.totalPromised - cost.Amount
}

// GetBalance returns the current balance
func (bt *BalanceTracker) GetBalance() uint64 {
	bt.calculateBalance()
	return bt.balance
}

// Start starts keeping track of time for balance
func (bt *BalanceTracker) Start() {
	bt.timeKeeper.StartTracking()
}

// Add increases the current balance by the given amount
func (bt *BalanceTracker) Add(amount uint64) {
	bt.Lock()
	defer bt.Unlock()
	bt.totalPromised += amount
}
