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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/money"
)

func Test_BalanceTracker(t *testing.T) {
	var initialBalance uint64 = 100
	mockTime := time.Second
	mockMoney := money.Money{
		Amount:   1,
		Currency: money.CurrencyMyst,
	}
	mtk := &mockTimeKeeper{elapsed: mockTime}
	mac := &mockAmountCalculator{toReturn: mockMoney}
	tracker := NewBalanceTracker(mtk, mac, initialBalance)
	balance := tracker.GetBalance()

	balanceAfter := initialBalance - mockMoney.Amount
	assert.Equal(t, balanceAfter, tracker.balance)
	assert.Equal(t, balanceAfter, balance)
	assert.Equal(t, mac.calledWith, mockTime)

	assert.False(t, mtk.startCalled)

	tracker.Start()
	assert.True(t, mtk.startCalled)

	var promisedAmount uint64 = 1
	tracker.Add(promisedAmount)
	assert.Equal(t, tracker.totalPromised, promisedAmount+initialBalance)
}

type mockTimeKeeper struct {
	elapsed     time.Duration
	startCalled bool
}

func (mtk *mockTimeKeeper) StartTracking() {
	mtk.startCalled = true
}

func (mtk *mockTimeKeeper) Elapsed() time.Duration {
	return mtk.elapsed
}

type mockAmountCalculator struct {
	calledWith time.Duration
	toReturn   money.Money
}

func (mac *mockAmountCalculator) TotalAmount(duration time.Duration) money.Money {
	mac.calledWith = duration
	return mac.toReturn
}
