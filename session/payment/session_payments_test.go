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

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
)

type MockPeerPromiseSender struct {
	mockError     error
	chanToWriteTo chan promise.PromiseMessage
}

func (mpps *MockPeerPromiseSender) Send(p promise.PromiseMessage) error {
	if mpps.chanToWriteTo != nil {
		mpps.chanToWriteTo <- p
	}
	return mpps.mockError
}

type MockPromiseTracker struct {
	promiseToReturn promises.IssuedPromise
	errToReturn     error
}

func (mpt *MockPromiseTracker) AlignStateWithProvider(providerState promise.State) error {
	// in this case we just do nothing really
	return nil
}

func (mpt *MockPromiseTracker) IssuePromiseWithAddedAmount(amountToAdd int64) (promises.IssuedPromise, error) {
	return mpt.promiseToReturn, mpt.errToReturn
}

var (
	balanceChannel  = make(chan balance.Message, 1)
	promiseToReturn = promises.IssuedPromise{
		Promise: promises.Promise{
			SeqNo:  1,
			Amount: 0,
		},
	}
	promiseSender  = &MockPeerPromiseSender{chanToWriteTo: make(chan promise.PromiseMessage, 1)}
	promiseTracker = &MockPromiseTracker{promiseToReturn: promiseToReturn, errToReturn: nil}
)

func NewTestSessionPayments(bm chan balance.Message, ps PeerPromiseSender, pt PromiseTracker) *SessionPayments {
	return NewSessionPayments(
		bm,
		ps,
		pt,
	)
}

func Test_SessionPayments_Start_Stop(t *testing.T) {
	cpo := NewTestSessionPayments(balanceChannel, promiseSender, promiseTracker)
	go func() {
		time.Sleep(time.Nanosecond * 10)
		cpo.Stop()
	}()
	err := cpo.Start()
	assert.Nil(t, err)
}

func Test_SessionPayments_SendsPromiseOnBalance(t *testing.T) {
	cpo := NewTestSessionPayments(balanceChannel, promiseSender, promiseTracker)
	go func() { cpo.Start() }()
	defer cpo.Stop()
	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	for v := range promiseSender.chanToWriteTo {
		assert.Exactly(t, promise.PromiseMessage{SequenceID: 1, Amount: 0, Signature: "0x"}, v)
		break
	}
}

func Test_SessionPayments_ReportsIssuingErrors(t *testing.T) {
	customTracker := *promiseTracker
	err := errors.New("issuing failed")
	customTracker.errToReturn = err
	cpo := NewTestSessionPayments(balanceChannel, promiseSender, &customTracker)

	go func() {
		err := cpo.Start()
		assert.Equal(t, customTracker.errToReturn, err)
	}()

	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
}

func Test_SessionPayments_ReportsSendingErrors(t *testing.T) {
	customSender := *promiseSender
	err := errors.New("sending failed")
	customSender.mockError = err

	cpo := NewTestSessionPayments(balanceChannel, &customSender, promiseTracker)
	go func() {
		err := cpo.Start()
		assert.Equal(t, customSender.mockError, err)
	}()

	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
}
