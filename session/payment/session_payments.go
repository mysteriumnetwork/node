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
}

// NewSessionPayments returns a new instance of consumer payment orchestrator
func NewSessionPayments(balanceChan chan balance.Message, peerPromiseSender PeerPromiseSender, promiseTracker PromiseTracker) *SessionPayments {
	return &SessionPayments{
		stop:              make(chan struct{}),
		balanceChan:       balanceChan,
		peerPromiseSender: peerPromiseSender,
		promiseTracker:    promiseTracker,
	}
}

// Start starts the payment orchestrator. Blocks.
func (cpo *SessionPayments) Start() error {
	for {
		select {
		case <-cpo.stop:
			return nil
		case balance := <-cpo.balanceChan:
			err := cpo.promiseTracker.AlignStateWithProvider(promise.State{
				Seq:    balance.SequenceID,
				Amount: balance.Balance,
			})
			if err != nil {
				return err
			}
			issuedPromise, err := cpo.promiseTracker.ExtendPromise(balance.Balance)
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
		}
	}
}

// Stop stops the payment orchestrator
func (cpo *SessionPayments) Stop() {
	close(cpo.stop)
}
