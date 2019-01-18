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

// Package payment is responsible for ensuring that the consumer can fullfil his obligation to provider.
// It contains all the orchestration required for value transfer from consumer to provider.
package payment

import (
	"errors"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
)

// BalanceTracker keeps track of current balance
type BalanceTracker interface {
	GetBalance() balance.Message
}

// PeerPromiseReceiver receives promises from peer
type PeerPromiseReceiver interface {
	Listen() <-chan promise.PromiseMessage
}

// PromiseValidator validates given promise
type PromiseValidator interface {
	Validate(promise.PromiseMessage) bool
}

// PeerBalanceSender knows how to send a balance message to the peer
type PeerBalanceSender interface {
	Send(balance.Message) error
}

// ErrPromiseWaitTimeout indicates that we waited for a promise long enough, but with no result
var ErrPromiseWaitTimeout = errors.New("Did not get a new promise")

// ErrPromiseValidationFailed indicates that an invalid promise was sent
var ErrPromiseValidationFailed = errors.New("Promise validation failed")

// ProviderPaymentOrchestrator orchestrates the ping pong of balance sent to consumer -> promise received from consumer flow
type ProviderPaymentOrchestrator struct {
	stop                chan struct{}
	peerBalanceSender   PeerBalanceSender
	balanceTracker      BalanceTracker
	peerPromiseReceiver PeerPromiseReceiver
	period              time.Duration
	promiseWaitTimeout  time.Duration
	promiseValidator    PromiseValidator
}

// NewProviderPaymentOrchestrator creates a new instance of provider payment orchestrator
func NewProviderPaymentOrchestrator(
	peerBalanceSender PeerBalanceSender,
	balanceTracker BalanceTracker,
	peerPromiseReceiver PeerPromiseReceiver,
	period time.Duration,
	promiseWaitTimeout time.Duration,
	promiseValidator PromiseValidator) *ProviderPaymentOrchestrator {
	return &ProviderPaymentOrchestrator{
		stop:                make(chan struct{}, 1),
		peerBalanceSender:   peerBalanceSender,
		balanceTracker:      balanceTracker,
		peerPromiseReceiver: peerPromiseReceiver,
		period:              period,
		promiseWaitTimeout:  promiseWaitTimeout,
		promiseValidator:    promiseValidator,
	}
}

// Start starts the payment orchestrator. Returns a read only channel that indicates if any errors are encountered.
// The channel is closed when the orchestrator is stopped.
func (ppo *ProviderPaymentOrchestrator) Start() <-chan error {
	ch := make(chan error, 1)
	listenChannel := ppo.peerPromiseReceiver.Listen()

	go func() {
		defer close(ch)
		for {
			select {
			case <-ppo.stop:
				return
			case <-time.After(ppo.period):
				ppo.sendBalance(ch)
				ppo.receivePromiseOrTimeout(listenChannel, ch)
			}
		}
	}()

	return ch
}

func (ppo *ProviderPaymentOrchestrator) sendBalance(ch chan error) {
	balance := ppo.balanceTracker.GetBalance()
	// TODO: Maybe retry a couple of times?
	err := ppo.peerBalanceSender.Send(balance)
	if err != nil {
		ch <- err
	}
}

func (ppo *ProviderPaymentOrchestrator) receivePromiseOrTimeout(listenChannel <-chan promise.PromiseMessage, errorChan chan error) {
	select {
	case pm := <-listenChannel:
		log.Info("Promise received", pm)
		if !ppo.promiseValidator.Validate(pm) {
			errorChan <- ErrPromiseValidationFailed
		}
		// TODO: Save the promise
		// TODO: Change balance
	case <-time.After(ppo.promiseWaitTimeout):
		errorChan <- ErrPromiseWaitTimeout
	}
}

// Stop stops the payment orchestrator
func (ppo *ProviderPaymentOrchestrator) Stop() {
	ppo.stop <- struct{}{}
}
