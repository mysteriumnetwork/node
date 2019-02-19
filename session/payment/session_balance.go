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

	sequenceID uint64
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
func (sb *SessionBalance) Start() error {
	lastPromise, err := sb.loadInitialPromiseState()
	if err != nil {
		return err
	}

	sb.startBalanceTracker(lastPromise)

	for {
		select {
		case <-sb.stop:
			return nil
		case <-time.After(sb.chargePeriod):
			err := sb.sendBalance()
			if err != nil {
				return err
			}
			err = sb.receivePromiseOrTimeout()
			if err != nil {
				return err
			}
		}
	}
}

func (sb *SessionBalance) loadInitialPromiseState() (promise.StoredPromise, error) {
	lastPromise, err := sb.promiseStorage.GetLastPromise(sb.issuer)
	if err != nil {
		// if an error occurs when fetching the last promise, issue a new id
		sb.sequenceID, err = sb.promiseStorage.GetNewSeqIDForIssuer(sb.issuer)
		return lastPromise, err
	}
	sb.sequenceID = lastPromise.SequenceID
	return lastPromise, nil
}

func (sb *SessionBalance) startBalanceTracker(lastPromise promise.StoredPromise) {
	amountToAdd := lastPromise.UnconsumedAmount

	sb.balanceTracker.Add(amountToAdd)
	sb.balanceTracker.Start()
}

func (sb *SessionBalance) sendBalance() error {
	currentBalance := sb.balanceTracker.GetBalance()
	p, err := sb.promiseStorage.GetLastPromise(sb.issuer)
	if err != nil {
		return err
	}

	// if we're ever in a situation where the unconsumed amount is zero, but the balance is not - something is definitely not right
	if p.UnconsumedAmount == 0 && currentBalance != 0 {
		return fmt.Errorf("unconsumed amount is 0, while balance is %v", currentBalance)
	}

	err = sb.promiseStorage.Update(sb.issuer, promise.StoredPromise{
		SequenceID:       p.SequenceID,
		UnconsumedAmount: currentBalance,
		Message:          p.Message,
		AddedAt:          p.AddedAt,
	})
	if err != nil {
		return err
	}

	// TODO: figure out when to get a new sequenceID
	return sb.peerBalanceSender.Send(balance.Message{
		Balance:    currentBalance,
		SequenceID: sb.sequenceID,
	})
}

func (sb *SessionBalance) calculateAmountToAdd(pm promise.Message, p promise.StoredPromise) uint64 {
	var amountToSubtract uint64
	if p.Message != nil {
		amountToSubtract = p.Message.Amount
	}
	amountToAdd := pm.Amount - amountToSubtract
	return amountToAdd
}

func (sb *SessionBalance) storePromiseAndUpdateBalance(pm promise.Message) error {
	p, err := sb.promiseStorage.GetLastPromise(sb.issuer)
	if err != nil {
		return err
	}
	amount := sb.calculateAmountToAdd(pm, p)
	sb.balanceTracker.Add(amount)

	err = sb.promiseStorage.Update(sb.issuer, promise.StoredPromise{
		SequenceID:       pm.SequenceID,
		Message:          &pm,
		UnconsumedAmount: p.UnconsumedAmount + amount,
		AddedAt:          p.AddedAt,
	})
	return err
}

func (sb *SessionBalance) receivePromiseOrTimeout() error {
	select {
	case pm := <-sb.promiseChan:
		if !sb.promiseValidator.Validate(pm) {
			return ErrPromiseValidationFailed
		}

		// TODO: check for consumer sending fishy sequenceIDs and amounts
		err := sb.storePromiseAndUpdateBalance(pm)
		if err != nil {
			return err
		}
	case <-time.After(sb.promiseWaitTimeout):
		return ErrPromiseWaitTimeout
	}
	return nil
}

// Stop stops the payment orchestrator
func (sb *SessionBalance) Stop() {
	close(sb.stop)
}
