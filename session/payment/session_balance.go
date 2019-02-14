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
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
)

// PromiseStorage stores the promises and issues new sequenceID's
type PromiseStorage interface {
	GetNewSeqIDForIssuer(issuerID identity.Identity) (uint64, error)
	Update(issuerID identity.Identity, promise promise.StoredPromise) error
	GetLastPromise(issuerID identity.Identity) (promise.StoredPromise, error)
}

// BalanceTracker keeps track of current balance
type BalanceTracker interface {
	GetBalance() uint64
	Start()
	Add(amount uint64)
}

// PromiseValidator validates given promise
type PromiseValidator interface {
	Validate(promise.Message) bool
}

// PeerBalanceSender knows how to send a balance message to the peer
type PeerBalanceSender interface {
	Send(balance.Message) error
}

// ErrPromiseWaitTimeout indicates that we waited for a promise long enough, but with no result
var ErrPromiseWaitTimeout = errors.New("did not get a new promise")

// ErrPromiseValidationFailed indicates that an invalid promise was sent
var ErrPromiseValidationFailed = errors.New("promise validation failed")

// SessionBalance orchestrates the ping pong of balance sent to consumer -> promise received from consumer flow
type SessionBalance struct {
	stop               chan struct{}
	peerBalanceSender  PeerBalanceSender
	balanceTracker     BalanceTracker
	promiseChan        chan promise.Message
	chargePeriod       time.Duration
	promiseWaitTimeout time.Duration
	promiseValidator   PromiseValidator
	promiseStorage     PromiseStorage
	issuer             identity.Identity

	sequenceID  uint64
	lastBalance uint64
}

// NewSessionBalance creates a new instance of provider payment orchestrator
func NewSessionBalance(
	peerBalanceSender PeerBalanceSender,
	balanceTracker BalanceTracker,
	promiseChan chan promise.Message,
	chargePeriod time.Duration,
	promiseWaitTimeout time.Duration,
	promiseValidator PromiseValidator,
	promiseStorage PromiseStorage,
	issuer identity.Identity) *SessionBalance {
	return &SessionBalance{
		stop:               make(chan struct{}),
		peerBalanceSender:  peerBalanceSender,
		balanceTracker:     balanceTracker,
		promiseChan:        promiseChan,
		chargePeriod:       chargePeriod,
		promiseWaitTimeout: promiseWaitTimeout,
		promiseValidator:   promiseValidator,
		promiseStorage:     promiseStorage,
		issuer:             issuer,
	}
}

// Start starts the payment orchestrator. Blocks.
func (ppo *SessionBalance) Start() error {
	lastPromise, err := ppo.loadInitialPromiseState()
	if err != nil {
		return err
	}

	ppo.startBalanceTracker(lastPromise)

	for {
		select {
		case <-ppo.stop:
			return nil
		case <-time.After(ppo.chargePeriod):
			err := ppo.sendBalance()
			if err != nil {
				return err
			}
			err = ppo.receivePromiseOrTimeout()
			if err != nil {
				return err
			}
		}
	}
}

func (ppo *SessionBalance) loadInitialPromiseState() (promise.StoredPromise, error) {
	lastPromise, err := ppo.promiseStorage.GetLastPromise(ppo.issuer)
	if err != nil {
		// if an error occurs when fetching the last promise, issue a new id
		ppo.sequenceID, err = ppo.promiseStorage.GetNewSeqIDForIssuer(ppo.issuer)
		if err != nil {
			return lastPromise, err
		}
	} else {
		ppo.sequenceID = lastPromise.SequenceID
	}
	return lastPromise, nil
}

func (ppo *SessionBalance) startBalanceTracker(lastPromise promise.StoredPromise) {
	amountToAdd := lastPromise.UnconsumedAmount

	ppo.balanceTracker.Add(amountToAdd)
	ppo.balanceTracker.Start()
}

func (ppo *SessionBalance) calculateUnconsumedPromiseAmount(balance uint64, lastPromise promise.StoredPromise) (uint64, error) {
	// if we're ever in a situation where the unconsumed amount is zero, but the balance is not - something is definitely not right
	if lastPromise.UnconsumedAmount == 0 && balance != 0 {
		return 0, fmt.Errorf("unconsumed amount is 0, while balance is %v", balance)
	}
	unconsumed := lastPromise.UnconsumedAmount - (lastPromise.UnconsumedAmount - balance)
	return unconsumed, nil
}

func (ppo *SessionBalance) sendBalance() error {
	b := ppo.balanceTracker.GetBalance()
	p, err := ppo.promiseStorage.GetLastPromise(ppo.issuer)
	if err != nil {
		return err
	}

	unconsumed, err := ppo.calculateUnconsumedPromiseAmount(b, p)
	if err != nil {
		return err
	}
	err = ppo.promiseStorage.Update(ppo.issuer, promise.StoredPromise{
		SequenceID:       p.SequenceID,
		UnconsumedAmount: unconsumed,
		Message:          p.Message,
		AddedAt:          p.AddedAt,
	})
	if err != nil {
		return err
	}

	// TODO: figure out when to get a new sequenceID
	return ppo.peerBalanceSender.Send(balance.Message{
		Balance:    b,
		SequenceID: ppo.sequenceID,
	})
}

func (ppo *SessionBalance) calculateAmountToAdd(pm promise.Message, p promise.StoredPromise) uint64 {
	var amountToSubtract uint64
	if p.Message != nil {
		amountToSubtract = p.Message.Amount
	}
	amountToAdd := pm.Amount - amountToSubtract
	return amountToAdd
}

func (ppo *SessionBalance) storePromiseAndUpdateBalance(pm promise.Message) error {
	p, err := ppo.promiseStorage.GetLastPromise(ppo.issuer)
	if err != nil {
		return err
	}
	amount := ppo.calculateAmountToAdd(pm, p)
	ppo.balanceTracker.Add(amount)

	err = ppo.promiseStorage.Update(ppo.issuer, promise.StoredPromise{
		SequenceID:       pm.SequenceID,
		Message:          &pm,
		UnconsumedAmount: p.UnconsumedAmount + amount,
		AddedAt:          p.AddedAt,
	})
	return err
}

func (ppo *SessionBalance) receivePromiseOrTimeout() error {
	select {
	case pm := <-ppo.promiseChan:
		if !ppo.promiseValidator.Validate(pm) {
			return ErrPromiseValidationFailed
		}

		// TODO: check for consumer sending fishy sequenceIDs and amounts
		err := ppo.storePromiseAndUpdateBalance(pm)
		if err != nil {
			return err
		}
	case <-time.After(ppo.promiseWaitTimeout):
		return ErrPromiseWaitTimeout
	}
	return nil
}

// Stop stops the payment orchestrator
func (ppo *SessionBalance) Stop() {
	close(ppo.stop)
}
