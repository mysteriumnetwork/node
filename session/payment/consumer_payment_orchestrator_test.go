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

func NewTestConsumerPaymentOrchestrator() *ConsumerPaymentOrchestrator {
	return NewConsumerPaymentOrchestrator(
		balanceChannel,
		promiseSender,
		promiseTracker,
	)
}

func Test_ConsumerPaymentOrchestrator_Start_Stop(t *testing.T) {
	cpo := NewTestConsumerPaymentOrchestrator()
	ch := cpo.Start()

	cpo.Stop()

	// read from channel to assert it is closed, test will timeout if we can't stop
	for range ch {
	}
}

func Test_ConsumerPaymentOrchestrator_SendsPromiseOnBalance(t *testing.T) {
	cpo := NewTestConsumerPaymentOrchestrator()
	_ = cpo.Start()
	defer cpo.Stop()
	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	for v := range promiseSender.chanToWriteTo {
		assert.Exactly(t, promise.PromiseMessage{SequenceID: 1, Amount: 0, Signature: "0x"}, v)
		break
	}
}

func Test_ConsumerPaymentOrchestrator_ReportsIssuingErrors(t *testing.T) {
	cpo := NewTestConsumerPaymentOrchestrator()
	ch := cpo.Start()
	defer cpo.Stop()

	err := errors.New("issuing failed")
	defer func() { promiseTracker.errToReturn = nil }()
	promiseTracker.errToReturn = err

	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	for v := range ch {
		assert.Equal(t, err, v)
		break
	}
}

func Test_ConsumerPaymentOrchestrator_ReportsSendingErrors(t *testing.T) {
	cpo := NewTestConsumerPaymentOrchestrator()
	ch := cpo.Start()
	defer cpo.Stop()

	err := errors.New("sending failed")
	defer func() { promiseSender.mockError = nil }()
	promiseSender.mockError = err

	balanceChannel <- balance.Message{Balance: 0, SequenceID: 1}
	for v := range ch {
		assert.Equal(t, err, v)
		break
	}
}
