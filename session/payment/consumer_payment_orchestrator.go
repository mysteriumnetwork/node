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
	Send(promise.PromiseMessage) error
}

type PromiseTracker interface {
	AlignStateWithProvider(providerState promise.State) error
	IssuePromiseWithAddedAmount(amountToAdd int64) (promises.IssuedPromise, error)
}

// ConsumerPaymentOrchestrator orchestrates the ping pong of balance received from provider -> promise sent to provider flow
type ConsumerPaymentOrchestrator struct {
	stop              chan struct{}
	balanceChan       chan balance.Message
	peerPromiseSender PeerPromiseSender
	promiseTracker    PromiseTracker
}

// NewConsumerPaymentOrchestrator returns a new instnace of consumer payment orchestrator
func NewConsumerPaymentOrchestrator(balanceChan chan balance.Message, peerPromiseSender PeerPromiseSender, promiseTracker PromiseTracker) *ConsumerPaymentOrchestrator {
	return &ConsumerPaymentOrchestrator{
		stop:              make(chan struct{}, 1),
		balanceChan:       balanceChan,
		peerPromiseSender: peerPromiseSender,
		promiseTracker:    promiseTracker,
	}
}

// Start starts the payment orchestrator. Returns a read only channel that indicates if any errors are encountered.
// The channel is closed when the orchestrator is stopped.
func (cpo *ConsumerPaymentOrchestrator) Start() <-chan error {
	errorChannel := make(chan error, 1)

	go func() {
		defer close(errorChannel)
		for {
			select {
			case <-cpo.stop:
				return
			case balance := <-cpo.balanceChan:
				err := cpo.promiseTracker.AlignStateWithProvider(promise.State{
					// TODO: figure out the int64/uint64 mess
					Seq:    int64(balance.SequenceID),
					Amount: int64(balance.Balance),
				})
				if err != nil {
					errorChannel <- err
					return
				}
				// TODO: figure out the int64/uint64 mess
				issuedPromise, err := cpo.promiseTracker.IssuePromiseWithAddedAmount(int64(balance.Balance))
				if err != nil {
					errorChannel <- err
					return
				}
				err = cpo.peerPromiseSender.Send(promise.PromiseMessage{
					Amount:     uint64(issuedPromise.Promise.Amount),
					SequenceID: uint64(issuedPromise.Promise.SeqNo),
					Signature:  fmt.Sprintf("0x%v", hex.EncodeToString(issuedPromise.IssuerSignature)),
				})
				if err != nil {
					errorChannel <- err
					return
				}
			}
		}
	}()

	return errorChannel
}

// Stop stops the payment orchestrator
func (cpo *ConsumerPaymentOrchestrator) Stop() {
	cpo.stop <- struct{}{}
}
