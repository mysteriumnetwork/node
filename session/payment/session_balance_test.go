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

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
)

var (
	issuer              = identity.FromAddress("0x0")
	consumer            = identity.FromAddress("0x00")
	receiver            = identity.FromAddress("0x000")
	MBT                 = &MockBalanceTracker{balanceToReturn: 0}
	MPV                 = &MockPromiseValidator{isValid: true}
	mockPromiseToReturn = promise.StoredPromise{
		SequenceID: 1,
		ConsumerID: consumer,
	}
	MPS = &MockPromiseStorage{
		promiseForConsumerToReturn: mockPromiseToReturn,
	}
)

func newMockPeerBalanceSender() *MockPeerBalanceSender {
	return &MockPeerBalanceSender{balanceMessages: make(chan balance.Message)}
}

func NewMockSessionBalance(balanceSender *MockPeerBalanceSender, mpv *MockPromiseValidator, mps *MockPromiseStorage, mbt *MockBalanceTracker) *SessionBalance {
	return NewSessionBalance(
		balanceSender,
		mbt,
		make(chan promise.Message),
		time.Millisecond*1,
		time.Millisecond*1,
		mpv,
		mps,
		consumer,
		receiver,
		issuer,
	)
}

func Test_calculateMaxNotReceivedPromiseCount(t *testing.T) {
	res := calculateMaxNotReceivedPromiseCount(time.Minute*5, time.Second*240)
	assert.Equal(t, uint64(1), res)
	res = calculateMaxNotReceivedPromiseCount(time.Minute*5, time.Second*20)
	assert.Equal(t, uint64(15), res)
	res = calculateMaxNotReceivedPromiseCount(time.Hour*2, time.Second*20)
	assert.Equal(t, uint64(360), res)
}

func Test_SessionBalanceStartStop(t *testing.T) {
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, MPS, MBT)
	go func() {
		orch.Stop()
	}()
	err := orch.Start()
	assert.Nil(t, err)
}

func Test_SessionBalanceSendsBalance(t *testing.T) {
	bs := newMockPeerBalanceSender()
	orch := NewMockSessionBalance(bs, MPV, MPS, MBT)
	defer orch.Stop()
	go orch.Start()

	assert.Exactly(t, balance.Message{SequenceID: 1, Balance: 0}, <-bs.balanceMessages)
}

func Test_SessionBalanceSendsBalance_Timeouts(t *testing.T) {
	bs := newMockPeerBalanceSender()
	orch := NewMockSessionBalance(bs, MPV, MPS, MBT)
	defer orch.Stop()

	// add a shorter timeout
	orch.promiseWaitTimeout = time.Nanosecond

	// and define the threshold since the default is going to be way too big
	orch.maxNotReceivedPromises = 2

	testDone := make(chan struct{})
	go func() {
		err := orch.Start()
		assert.Equal(t, ErrPromiseWaitTimeout, err)
		testDone <- struct{}{}
	}()

	<-bs.balanceMessages
	<-bs.balanceMessages

	<-testDone
}

func Test_SessionBalanceSendsBalance_DoesNotTimeoutIfReceivesAPromise(t *testing.T) {
	bs := newMockPeerBalanceSender()
	orch := NewMockSessionBalance(bs, MPV, MPS, MBT)
	// define the threshold since the default is going to be way too big
	orch.maxNotReceivedPromises = 3

	testDone := make(chan struct{})
	go func() {
		err := orch.Start()
		assert.Nil(t, err)
		testDone <- struct{}{}
	}()

	<-bs.balanceMessages
	<-bs.balanceMessages
	orch.promiseChan <- promise.Message{
		Amount:     100,
		SequenceID: 1,
		Signature:  "0x1111",
	}

	orch.Stop()

	<-testDone
	assert.Equal(t, uint64(0), orch.notReceivedPromiseCount)
}

func Test_SessionBalanceInvalidPromise(t *testing.T) {
	mpv := *MPV
	mpv.isValid = false

	bs := newMockPeerBalanceSender()

	orch := NewMockSessionBalance(bs, &mpv, MPS, MBT)
	defer orch.Stop()

	testDone := make(chan struct{})
	go func() {
		err := orch.Start()
		assert.Equal(t, ErrPromiseValidationFailed, err)
		testDone <- struct{}{}
	}()

	<-bs.balanceMessages
	orch.promiseChan <- promise.Message{
		Amount:     100,
		SequenceID: 1,
		Signature:  "0x1111",
	}

	// TODO: need a happy path test for this.
	<-testDone
}

func Test_SessionBalance_LoadInitialPromiseState_WithExistingPromise(t *testing.T) {
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, MPS, MBT)
	promise, err := orch.loadInitialPromiseState()
	assert.Nil(t, err)
	assert.Equal(t, MPS.promiseForConsumerToReturn, promise)
	assert.Equal(t, MPS.promiseForConsumerToReturn.SequenceID, orch.sequenceID)
}

func Test_SessionBalance_LoadInitialPromiseState_WithoutExistingPromise(t *testing.T) {
	mps := *MPS
	mps.promiseForConsumerError = errBoltNotFound
	mps.promiseForConsumerToReturn = promise.StoredPromise{}
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, &mps, MBT)
	promise, err := orch.loadInitialPromiseState()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), promise.SequenceID)
}

func Test_SessionBalance_LoadInitialPromiseState_WithDifferentConsumer_IssuesNew(t *testing.T) {
	mps := *MPS
	mps.promiseForConsumerToReturn = promise.StoredPromise{
		SequenceID: 3,
	}
	mps.promiseForConsumerError = errNoPromiseForConsumer
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, &mps, MBT)
	promise, err := orch.loadInitialPromiseState()
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), promise.SequenceID)
}

func Test_SessionBalance_LoadInitialPromiseState_BubblesErrors(t *testing.T) {
	mps := *MPS
	mps.promiseForConsumerError = errors.New("test")
	mps.newIDerror = mps.promiseForConsumerError
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, &mps, MBT)
	_, err := orch.loadInitialPromiseState()
	assert.Equal(t, mps.promiseForConsumerError, err)
}

func Test_SessionBalance_StartBalanceTracker_AddsUnconsumedAmount(t *testing.T) {
	mbt := *MBT
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, MPS, &mbt)
	lp := promise.StoredPromise{UnconsumedAmount: 100}
	orch.startBalanceTracker(lp)
	assert.Equal(t, lp.UnconsumedAmount, mbt.amountAdded)
}

func Test_SessionBalance_CalculateAmountToAdd(t *testing.T) {
	orch := NewMockSessionBalance(newMockPeerBalanceSender(), MPV, MPS, MBT)
	lp := promise.StoredPromise{}
	msg := promise.Message{
		Amount: 100,
	}
	amount := orch.calculateAmountToAdd(msg, lp)
	assert.Equal(t, msg.Amount, amount)

	lp = promise.StoredPromise{
		Message: &promise.Message{
			Amount: 50,
		},
	}
	amount = orch.calculateAmountToAdd(msg, lp)
	assert.Equal(t, lp.Message.Amount, amount)

	lp = promise.StoredPromise{
		Message: &promise.Message{
			Amount: 100,
		},
	}
	amount = orch.calculateAmountToAdd(msg, lp)
	assert.Equal(t, uint64(0), amount)

}

type MockPromiseStorage struct {
	promiseForConsumerToReturn promise.StoredPromise
	promiseForConsumerError    error
	newIDerror                 error
	updateError                error
}

func (mps *MockPromiseStorage) GetNewSeqIDForIssuer(consumerID, receiverID, issuerID identity.Identity) (uint64, error) {
	return mps.promiseForConsumerToReturn.SequenceID + 1, mps.newIDerror
}

func (mps *MockPromiseStorage) Update(issuerID identity.Identity, p promise.StoredPromise) error {
	return mps.updateError
}

func (mps *MockPromiseStorage) FindPromiseForConsumer(consumerID, receiverID, issuerID identity.Identity) (promise.StoredPromise, error) {
	return mps.promiseForConsumerToReturn, mps.promiseForConsumerError
}

type MockPeerBalanceSender struct {
	mockError       error
	balanceMessages chan balance.Message
}

func (mpbs *MockPeerBalanceSender) Send(b balance.Message) error {
	mpbs.balanceMessages <- b
	return mpbs.mockError
}

type MockBalanceTracker struct {
	balanceToReturn uint64
	amountAdded     uint64
	startCalled     bool
}

func (mbt *MockBalanceTracker) GetBalance() uint64 {
	return mbt.balanceToReturn
}

func (mbt *MockBalanceTracker) Add(amount uint64) {
	mbt.amountAdded = amount
}

func (mbt *MockBalanceTracker) Start() {
	mbt.startCalled = true
}

type MockPromiseValidator struct {
	isValid bool
}

func (mpv *MockPromiseValidator) Validate(promise.Message) bool {
	return mpv.isValid
}
