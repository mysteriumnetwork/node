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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
)

type MockPeerBalanceSender struct {
	mockError     error
	chanToWriteTo chan balance.Message
}

func (mpbs *MockPeerBalanceSender) Send(b balance.Message) error {
	if mpbs.chanToWriteTo != nil {
		mpbs.chanToWriteTo <- b
	}
	return mpbs.mockError
}

type MockPeerPromiseReceiver struct {
	lastBalance    balance.Message
	balanceChannel chan balance.Message
	promiseChannel chan promise.PromiseMessage
}

func NewMockPeerPromiseReceiver() *MockPeerPromiseReceiver {
	mpr := &MockPeerPromiseReceiver{
		balanceChannel: make(chan balance.Message, 1),
		promiseChannel: make(chan promise.PromiseMessage, 1),
	}
	go mpr.acceptBalances()
	return mpr
}

func (mppr *MockPeerPromiseReceiver) acceptBalances() {
	for v := range mppr.balanceChannel {
		mppr.lastBalance = v
		mppr.promiseChannel <- promise.PromiseMessage{
			SequenceID: v.SequenceID,
			Signature:  "test",
		}
	}
}

func (mppr *MockPeerPromiseReceiver) Listen() <-chan promise.PromiseMessage {
	return mppr.promiseChannel
}

type MockBalanceTracker struct {
	balanceMessage balance.Message
}

func (mbt *MockBalanceTracker) GetBalance() balance.Message {
	return mbt.balanceMessage
}

type MockPromiseValidator struct {
	isValid bool
}

func (mpv *MockPromiseValidator) Validate(promise.PromiseMessage) bool {
	return mpv.isValid
}

var (
	PromiseReceiver = NewMockPeerPromiseReceiver()
	BalanceSender   = &MockPeerBalanceSender{chanToWriteTo: PromiseReceiver.balanceChannel}
	MBT             = &MockBalanceTracker{balanceMessage: balance.Message{Balance: 0, SequenceID: 1}}
	MPV             = &MockPromiseValidator{isValid: true}
)

func NewMockProviderOrchestrator() *ProviderPaymentOrchestrator {
	return &ProviderPaymentOrchestrator{
		stop:                make(chan struct{}, 1),
		period:              time.Millisecond * 1,
		promiseWaitTimeout:  time.Millisecond * 1,
		peerBalanceSender:   BalanceSender,
		peerPromiseReceiver: PromiseReceiver,
		balanceTracker:      MBT,
		promiseValidator:    MPV,
	}
}

func Test_ProviderPaymentOchestratorStartStop(t *testing.T) {
	orch := NewMockProviderOrchestrator()
	ch := orch.Start()
	orch.Stop()

	// read from channel to assert it is closed, test will timeout if we can't stop
	for range ch {
	}
}

func Test_ProviderPaymentOchestratorSendsBalance(t *testing.T) {
	orch := NewMockProviderOrchestrator()
	defer orch.Stop()
	_ = orch.Start()

	time.Sleep(time.Millisecond * 2)
	assert.Exactly(t, balance.Message{SequenceID: 1, Balance: 0}, PromiseReceiver.lastBalance)
}

func Test_ProviderPaymentOchestratorSendsBalance_Timeouts(t *testing.T) {
	orch := NewMockProviderOrchestrator()
	defer orch.Stop()

	// add a shorter timeout
	orch.promiseWaitTimeout = time.Nanosecond
	ch := orch.Start()

	for v := range ch {
		assert.Equal(t, ErrPromiseWaitTimeout, v)
		break
	}
}

func Test_ProviderPaymentOchestratorInvalidPromise(t *testing.T) {
	orch := NewMockProviderOrchestrator()
	defer orch.Stop()

	MPV.isValid = false
	defer func() {
		MPV.isValid = true
	}()

	ch := orch.Start()

	for v := range ch {
		assert.Equal(t, ErrPromiseValidationFailed, v)
		break
	}
}
