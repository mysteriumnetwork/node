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

	log "github.com/cihub/seelog"
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
func NewProviderBalanceTracker(sender PeerSender, timeKeeper TimeKeeper, amountCalculator AmountCalculator, period time.Duration, initialBalance uint64) *ProviderBalanceTracker {
	return &ProviderBalanceTracker{
		sender:           sender,
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

func (pbt *ProviderBalanceTracker) periodicSend() error {
	for {
		select {
		case <-pbt.stop:
			return nil
		case <-time.After(pbt.period):
			pbt.calculateBalance()
			// TODO: Maybe retry a couple of times?
			err := pbt.sendMessage()
			if err != nil {
				log.Error(balanceTrackerPrefix, "Balance tracker failed to send the balance message")
				log.Error(balanceTrackerPrefix, err)
			}
			// TODO: destroy session/connection if balance negative? or should we bubble the error and let the caller be responsible for this?
			// TODO: wait for response here on the promise topic
		}
	}
}

func (pbt *ProviderBalanceTracker) sendMessage() error {
	return pbt.sender.Send(pbt.getBalanceMessage())
}

func (pbt *ProviderBalanceTracker) getBalanceMessage() Message {
	// TODO: sequence ID should come here, somehow
	return Message{
		SequenceID: 0,
		Balance:    pbt.balance,
	}
}

// Track starts tracking the balance and sending it to the consumer
func (pbt *ProviderBalanceTracker) Track() error {
	pbt.timeKeeper.StartTracking()
	return pbt.periodicSend()
}

// Stop stops the balance tracker
func (pbt *ProviderBalanceTracker) Stop() {
	pbt.stop <- struct{}{}
}
