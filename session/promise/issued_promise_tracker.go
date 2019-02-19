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

package promise

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/promises"
)

// Issuer interface defines method to sign (issue) provided promise data and return promise with signature
// used by promise issuer (i.e. service consumer or 3d party)
type Issuer interface {
	Issue(promise promises.Promise) (promises.IssuedPromise, error)
}

// State defines current state of promise data (seq number and amount)
type State struct {
	Seq    uint64
	Amount uint64
}

// ConsumerTracker tracks and issues promises from consumer perspective, also validates states coming from service provider
type ConsumerTracker struct {
	current  State
	consumer identity.Identity
	receiver identity.Identity
	issuer   Issuer
}

// NewConsumerTracker returns the consumer side tracker for promises
func NewConsumerTracker(initial State, consumer, provider identity.Identity, issuer Issuer) *ConsumerTracker {
	return &ConsumerTracker{
		current:  initial,
		consumer: consumer,
		receiver: provider,
		issuer:   issuer,
	}
}

// ErrUnexpectedAmount represents an error that occurs when we get an amount that's not aligned with our current understanding
var ErrUnexpectedAmount = errors.New("unexpected amount")

// AlignStateWithProvider aligns the consumers world with the providers
func (t *ConsumerTracker) AlignStateWithProvider(providerState State) error {
	if providerState.Seq > t.current.Seq {
		// new promise request
		t.current.Seq = providerState.Seq
		// ignore provider state value as new promise amount is always zero,
		// if provider tries to trick us to send more than expected it will be caught on next align
		t.current.Amount = 0
		return nil
	}
	if providerState.Amount > t.current.Amount {
		return ErrUnexpectedAmount
	}
	if providerState.Amount < t.current.Amount {
		return ErrUnexpectedAmount
	}

	return nil
}

// ExtendPromise issues a promise with the amount added to the promise
func (t *ConsumerTracker) ExtendPromise(amountToAdd uint64) (promises.IssuedPromise, error) {
	promise := promises.Promise{
		Extra: ExtraData{
			ConsumerAddress: common.HexToAddress(t.consumer.Address),
		},
		Receiver: common.HexToAddress(t.receiver.Address),
		Amount:   t.current.Amount + amountToAdd,
		SeqNo:    t.current.Seq,
	}
	t.current.Amount += amountToAdd
	return t.issuer.Issue(promise)
}
