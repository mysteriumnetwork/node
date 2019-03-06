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

package payment

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
)

// PeerPromiseSender knows how to send a promise message to the peer
type PeerPromiseSender interface {
	Send(promise.Message) error
}

// PromiseTracker keeps track of promises
type PromiseTracker interface {
	AlignStateWithProvider(providerState promise.State) error
	ExtendPromise(amountToAdd uint64) (promises.IssuedPromise, error)
}

// SessionPayments orchestrates the ping pong of balance received from provider -> promise sent to provider flow
type SessionPayments struct {
	stop              chan struct{}
	balanceChan       chan balance.Message
	peerPromiseSender PeerPromiseSender
	promiseTracker    PromiseTracker
	balanceTracker    BalanceTracker
}

// NewSessionPayments returns a new instance of consumer payment orchestrator
func NewSessionPayments(balanceChan chan balance.Message, peerPromiseSender PeerPromiseSender, promiseTracker PromiseTracker, balanceTracker BalanceTracker) *SessionPayments {
	return &SessionPayments{
		stop:              make(chan struct{}),
		balanceChan:       balanceChan,
		peerPromiseSender: peerPromiseSender,
		promiseTracker:    promiseTracker,
		balanceTracker:    balanceTracker,
	}
}

// balanceDifferenceThreshold determines the threshold where we'll cancel the session if there's a missmatch larger than the threshold provided between the provider and the consumer balances
const balanceDifferenceThreshold uint64 = 20

// ErrBalanceMissmatch represents an error that occurs when balances do not match
var ErrBalanceMissmatch = errors.New("balance missmatch")

// Start starts the payment orchestrator. Blocks.
func (cpo *SessionPayments) Start() error {
	cpo.balanceTracker.Start()
	for {
		select {
		case <-cpo.stop:
			return nil
		case balance := <-cpo.balanceChan:
			err := cpo.validateBalanceDifference(balance.Balance)
			if err != nil {
				return err
			}

			err = cpo.issuePromise(balance)
			if err != nil {
				return err
			}
		}
	}
}

func (cpo *SessionPayments) issuePromise(balance balance.Message) error {
	err := cpo.promiseTracker.AlignStateWithProvider(promise.State{
		Seq:    balance.SequenceID,
		Amount: balance.Balance,
	})
	if err != nil {
		return err
	}

	var amountToExtend uint64
	if balance.Balance == 0 {
		// TODO: this should probably not be hardcoded.
		amountToExtend = 100
	}
	issuedPromise, err := cpo.promiseTracker.ExtendPromise(amountToExtend)
	if err != nil {
		return err
	}
	err = cpo.peerPromiseSender.Send(promise.Message{
		Amount:     issuedPromise.Promise.Amount,
		SequenceID: issuedPromise.Promise.SeqNo,
		Signature:  fmt.Sprintf("0x%v", hex.EncodeToString(issuedPromise.IssuerSignature)),
	})
	if err != nil {
		return err
	}
	cpo.balanceTracker.Add(amountToExtend)
	return nil
}

func (cpo *SessionPayments) validateBalanceDifference(balance uint64) error {
	myBalance := cpo.balanceTracker.GetBalance()
	diff := calculateBalanceDifference(balance, myBalance)
	if diff >= balanceDifferenceThreshold {
		return ErrBalanceMissmatch
	}
	return nil
}

func calculateBalanceDifference(yourBalance, myBalance uint64) uint64 {
	if yourBalance > myBalance {
		return yourBalance - myBalance
	}
	return myBalance - yourBalance
}

// Stop stops the payment orchestrator
func (cpo *SessionPayments) Stop() {
	close(cpo.stop)
}
