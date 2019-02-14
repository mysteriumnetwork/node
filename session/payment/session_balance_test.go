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
	promiseChannel      = make(chan promise.Message)
	issuer              = identity.FromAddress("0x0")
	BalanceSender       = &MockPeerBalanceSender{balanceMessages: make(chan balance.Message)}
	MBT                 = &MockBalanceTracker{balanceMessage: balance.Message{Balance: 0, SequenceID: 1}}
	MPV                 = &MockPromiseValidator{isValid: true}
	mockPromiseToReturn = promise.StoredPromise{
		SequenceID: 1,
	}
	MPS = &MockPromiseStorage{promiseToReturn: mockPromiseToReturn}
)

func NewMockSessionBalance(mpv *MockPromiseValidator, mps *MockPromiseStorage, mbt *MockBalanceTracker) *SessionBalance {
	return NewSessionBalance(
		BalanceSender,
		mbt,
		promiseChannel,
		time.Millisecond*1,
		time.Millisecond*1,
		mpv,
		mps,
		issuer,
	)
}

func Test_ProviderPaymentOchestratorStartStop(t *testing.T) {
	orch := NewMockSessionBalance(MPV, MPS, MBT)
	go func() {
		orch.Stop()
	}()
	err := orch.Start()
	assert.Nil(t, err)
}

func Test_ProviderPaymentOchestratorSendsBalance(t *testing.T) {
	orch := NewMockSessionBalance(MPV, MPS, MBT)
	defer orch.Stop()
	go orch.Start()

	assert.Exactly(t, balance.Message{SequenceID: 1, Balance: 0}, <-BalanceSender.balanceMessages)
}

func Test_ProviderPaymentOchestratorSendsBalance_Timeouts(t *testing.T) {
	orch := NewMockSessionBalance(MPV, MPS, MBT)
	defer orch.Stop()

	// add a shorter timeout
	orch.promiseWaitTimeout = time.Nanosecond
	testDone := make(chan struct{})
	go func() {
		err := orch.Start()
		assert.Equal(t, ErrPromiseWaitTimeout, err)
		testDone <- struct{}{}
	}()

	//consume message but never respond
	<-BalanceSender.balanceMessages
	<-testDone
}

func Test_ProviderPaymentOchestratorInvalidPromise(t *testing.T) {
	mpv := *MPV
	mpv.isValid = false

	orch := NewMockSessionBalance(&mpv, MPS, MBT)
	defer orch.Stop()

	testDone := make(chan struct{})
	go func() {
		err := orch.Start()
		assert.Equal(t, ErrPromiseValidationFailed, err)
		testDone <- struct{}{}
	}()

	<-BalanceSender.balanceMessages
	promiseChannel <- promise.Message{
		Amount:     100,
		SequenceID: 1,
		Signature:  "0x1111",
	}

	// TODO: need a happy path test for this.
	<-testDone
}

func Test_SessionBalance_LoadInitialPromiseState_WithExistingPromise(t *testing.T) {
	orch := NewMockSessionBalance(MPV, MPS, MBT)
	promise, err := orch.loadInitialPromiseState()
	assert.Nil(t, err)
	assert.Equal(t, MPS.promiseToReturn, promise)
	assert.Equal(t, MPS.promiseToReturn.SequenceID, orch.sequenceID)
}

func Test_SessionBalance_LoadInitialPromiseState_WithoutExistingPromise(t *testing.T) {
	mps := *MPS
	mps.lastPromiseError = errors.New("test")
	orch := NewMockSessionBalance(MPV, &mps, MBT)
	promise, err := orch.loadInitialPromiseState()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), promise.SequenceID)
}

func Test_SessionBalance_LoadInitialPromiseState_BubblesErrors(t *testing.T) {
	mps := *MPS
	mps.lastPromiseError = errors.New("test")
	mps.newIDerror = mps.lastPromiseError
	orch := NewMockSessionBalance(MPV, &mps, MBT)
	_, err := orch.loadInitialPromiseState()
	assert.Equal(t, mps.lastPromiseError, err)
}

func Test_SessionBalance_StartBalanceTracker_AddsUnconsumedAmount(t *testing.T) {
	mbt := *MBT
	orch := NewMockSessionBalance(MPV, MPS, &mbt)
	lp := promise.StoredPromise{UnconsumedAmount: 100}
	orch.startBalanceTracker(lp)
	assert.Equal(t, lp.UnconsumedAmount, mbt.amountAdded)
}

func Test_SessionBalance_CalculateAmountToAdd(t *testing.T) {
	orch := NewMockSessionBalance(MPV, MPS, MBT)
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
	promiseToReturn  promise.StoredPromise
	newIDerror       error
	updateError      error
	lastPromiseError error
}

func (mps *MockPromiseStorage) GetNewSeqIDForIssuer(issuerID identity.Identity) (uint64, error) {
	return mps.promiseToReturn.SequenceID + 1, mps.newIDerror
}

func (mps *MockPromiseStorage) Update(issuerID identity.Identity, p promise.StoredPromise) error {
	return mps.updateError
}

func (mps *MockPromiseStorage) GetLastPromise(issuerID identity.Identity) (promise.StoredPromise, error) {
	return mps.promiseToReturn, mps.lastPromiseError
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
	balanceMessage balance.Message
	amountAdded    uint64
	startCalled    bool
}

func (mbt *MockBalanceTracker) GetBalance() uint64 {
	return mbt.balanceMessage.Balance
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
