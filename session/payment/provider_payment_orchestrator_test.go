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
	mockError       error
	balanceMessages chan balance.Message
}

func (mpbs *MockPeerBalanceSender) Send(b balance.Message) error {
	mpbs.balanceMessages <- b
	return mpbs.mockError
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
	promiseChannel = make(chan promise.PromiseMessage)
	BalanceSender  = &MockPeerBalanceSender{balanceMessages: make(chan balance.Message)}
	MBT            = &MockBalanceTracker{balanceMessage: balance.Message{Balance: 0, SequenceID: 1}}
	MPV            = &MockPromiseValidator{isValid: true}
)

func NewMockProviderOrchestrator() *ProviderPaymentOrchestrator {
	return NewProviderPaymentOrchestrator(
		BalanceSender,
		MBT,
		promiseChannel,
		time.Millisecond*1,
		time.Millisecond*1,
		MPV,
	)
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
	assert.Exactly(t, balance.Message{SequenceID: 1, Balance: 0}, <-BalanceSender.balanceMessages)
}

func Test_ProviderPaymentOchestratorSendsBalance_Timeouts(t *testing.T) {
	orch := NewMockProviderOrchestrator()
	defer orch.Stop()

	// add a shorter timeout
	orch.promiseWaitTimeout = time.Nanosecond
	ch := orch.Start()

	//consume message but never respond
	<-BalanceSender.balanceMessages

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

	<-BalanceSender.balanceMessages
	promiseChannel <- promise.PromiseMessage{
		Amount:     100,
		SequenceID: 1,
		Signature:  "0x1111",
	}

	for v := range ch {
		assert.Equal(t, ErrPromiseValidationFailed, v)
		break
	}
}
