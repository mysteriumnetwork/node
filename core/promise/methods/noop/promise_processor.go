/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

const (
	balanceNotifying = balanceState("Notifying")
	balanceStopped   = balanceState("Stopped")
)

// NewPromiseProcessor creates instance of PromiseProcessor
func NewPromiseProcessor(dialog communication.Dialog, balance identity.Balance, storage promise.Storer) *PromiseProcessor {
	return &PromiseProcessor{
		dialog:  dialog,
		balance: balance,
		storage: storage,

		balanceInterval: 5 * time.Second,
		balanceState:    balanceStopped,
		balanceShutdown: make(chan bool, 1),
	}
}

type balanceState string

// PromiseProcessor process promises in such way, what no actual money is deducted from promise
type PromiseProcessor struct {
	dialog  communication.Dialog
	balance identity.Balance
	storage promise.Storer

	balanceInterval   time.Duration
	balanceState      balanceState
	balanceStateMutex sync.RWMutex
	balanceShutdown   chan bool

	// these are populated later at runtime
	lastPromise promise.Promise
}

// Start processing promises for given service proposal
func (processor *PromiseProcessor) Start(proposal market.ServiceProposal) error {
	// TODO: replace static value with some real data
	processor.lastPromise = promise.Promise{
		Amount: money.NewMoney(10, money.CurrencyMyst),
	}

	consumer := promise.NewConsumer(proposal, processor.balance, processor.storage)
	if err := processor.dialog.Respond(consumer); err != nil {
		return err
	}

	processor.balanceShutdown = make(chan bool, 1)
	go processor.balanceLoop()

	return nil
}

// Stop stops processing promises
func (processor *PromiseProcessor) Stop() error {
	processor.balanceShutdown <- true
	return nil
}

func (processor *PromiseProcessor) balanceLoop() {
	processor.setBalanceState(balanceNotifying)

balanceLoop:
	for {
		select {
		case <-processor.balanceShutdown:
			break balanceLoop

		case <-time.After(processor.balanceInterval):
			// TODO: replace static value with some real data
			processor.balanceSend(
				promise.BalanceMessage{RequestID: 1, Accepted: true, Balance: processor.lastPromise.Amount},
			)
		}
	}

	processor.setBalanceState(balanceStopped)
}

func (processor *PromiseProcessor) setBalanceState(state balanceState) {
	processor.balanceStateMutex.Lock()
	defer processor.balanceStateMutex.Unlock()

	processor.balanceState = state
}

func (processor *PromiseProcessor) getBalanceState() balanceState {
	processor.balanceStateMutex.RLock()
	defer processor.balanceStateMutex.RUnlock()

	return processor.balanceState
}

func (processor *PromiseProcessor) balanceSend(message promise.BalanceMessage) error {
	log.Infof("notifying balance %s", message.Balance.String())
	return processor.dialog.Send(&promise.BalanceMessageProducer{
		Message: message,
	})
}
