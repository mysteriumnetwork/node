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
	"errors"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/model"
	"github.com/stretchr/testify/assert"
)

type MockPeerPromiseSender struct {
	mockError     error
	chanToWriteTo chan promise.Message
}

func (mpps *MockPeerPromiseSender) Send(p promise.Message) error {
	if mpps.chanToWriteTo != nil {
		mpps.chanToWriteTo <- p
	}
	return mpps.mockError
}

type MockPromiseTracker struct {
	promiseToReturn model.IssuedPromise
	errToReturn     error
}

func (mpt *MockPromiseTracker) AlignStateWithProvider(providerState promise.State) error {
	// in this case we just do nothing really
	return nil
}

func (mpt *MockPromiseTracker) ExtendPromise(amountToAdd uint64) (model.IssuedPromise, error) {
	return mpt.promiseToReturn, mpt.errToReturn
}

var (
	promiseToReturn = model.IssuedPromise{
		Promise: model.Promise{
			SeqNo:  1,
			Amount: 0,
		},
	}
	promiseTracker = &MockPromiseTracker{promiseToReturn: promiseToReturn, errToReturn: nil}
	balanceTracker = &MockBalanceTracker{balanceToReturn: 0}
)

func newPromiseSender() *MockPeerPromiseSender {
	return &MockPeerPromiseSender{chanToWriteTo: make(chan promise.Message, 1)}
}

func NewTestSessionPayments(bm chan balance.Message, ps PeerPromiseSender, pt PromiseTracker, bt BalanceTracker) *SessionPayments {
	return NewSessionPayments(
		bm,
		ps,
		pt,
		bt,
	)
}

func Test_SessionPayments_Start_Stop(t *testing.T) {
	cpo := NewTestSessionPayments(make(chan balance.Message, 1), newPromiseSender(), promiseTracker, balanceTracker)
	go func() {
		time.Sleep(time.Nanosecond * 10)
		cpo.Stop()
	}()
	err := cpo.Start()
	assert.Nil(t, err)
}

func Test_SessionPayments_SendsPromiseOnBalance(t *testing.T) {
	balanceChannel := make(chan balance.Message, 1)
	promiseSender := newPromiseSender()
	cpo := NewTestSessionPayments(balanceChannel, promiseSender, promiseTracker, balanceTracker)
	go cpo.Start()
	defer cpo.Stop()
	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	for v := range promiseSender.chanToWriteTo {
		assert.Exactly(t, promise.Message{SequenceID: 1, Amount: 0, Signature: "0x"}, v)
		break
	}
}

func Test_SessionPayments_ReportsIssuingErrors(t *testing.T) {
	balanceChannel := make(chan balance.Message, 1)
	customTracker := *promiseTracker
	err := errors.New("issuing failed")
	customTracker.errToReturn = err
	cpo := NewTestSessionPayments(balanceChannel, newPromiseSender(), &customTracker, balanceTracker)

	testDone := make(chan struct{})
	go func() {
		err := cpo.Start()
		assert.Equal(t, customTracker.errToReturn, err)
		testDone <- struct{}{}
	}()

	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	<-testDone
}

func Test_SessionPayments_ErrsOnBalanceMissmatch(t *testing.T) {
	balanceChannel := make(chan balance.Message, 1)
	cpo := NewTestSessionPayments(balanceChannel, newPromiseSender(), promiseTracker, balanceTracker)
	testDone := make(chan struct{})

	go func() {
		err := cpo.Start()
		assert.Equal(t, ErrBalanceMissmatch, err)
		testDone <- struct{}{}
	}()

	balanceChannel <- balance.Message{Balance: 100, SequenceID: 1}
	<-testDone
}

func Test_SessionPayments_NoPanicOnSecondStop(t *testing.T) {
	balanceChannel := make(chan balance.Message, 1)
	promiseSender := newPromiseSender()
	cpo := NewTestSessionPayments(balanceChannel, promiseSender, promiseTracker, balanceTracker)
	go cpo.Start()
	cpo.Stop()
	cpo.Stop()
}
