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
package pingpong

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestPromiseSettler_loadInitialState(t *testing.T) {
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()

	settler := NewHermesPromiseSettler(&mockTransactor{}, &mockHermesChannelProvider{}, &mockProviderChannelStatusProvider{}, mrsp, ks, &settlementHistoryStorageMock{}, cfg)
	settler.currentState[mockID] = settlementState{}

	// check if existing gets skipped
	err := settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v := settler.currentState[mockID]
	assert.EqualValues(t, settlementState{}, v)

	// check if unregistered gets skipped
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.Registered,
	}

	err = settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	assert.EqualValues(t, settlementState{
		registered: true,
	}, v)

	// check if will resync
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.Registered,
	}

	err = settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	assert.True(t, v.registered)

	// check if will bubble registration status errors
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.Registered,
		err:    errMock,
	}

	err = settler.loadInitialState(mockID)
	assert.Equal(t, fmt.Sprintf("could not check registration status for %v: %v", mockID, errMock.Error()), err.Error())
}

func TestPromiseSettler_handleRegistrationEvent(t *testing.T) {
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()
	settler := NewHermesPromiseSettler(&mockTransactor{}, &mockHermesChannelProvider{}, &mockProviderChannelStatusProvider{}, mrsp, ks, &settlementHistoryStorageMock{}, cfg)

	statusesWithNoChangeExpected := []registry.RegistrationStatus{registry.Unregistered, registry.InProgress, registry.RegistrationError}
	for _, v := range statusesWithNoChangeExpected {
		settler.handleRegistrationEvent(registry.AppEventIdentityRegistration{
			ID:     mockID,
			Status: v,
		})

		_, ok := settler.currentState[mockID]

		assert.False(t, ok)
	}

	settler.handleRegistrationEvent(registry.AppEventIdentityRegistration{
		ID:     mockID,
		Status: registry.Registered,
	})

	v, ok := settler.currentState[mockID]
	assert.True(t, ok)
	assert.True(t, v.registered)
}

func TestPromiseSettler_handleHermesPromiseReceived(t *testing.T) {
	channelProvider := &mockHermesChannelProvider{}
	channelStatusProvider := &mockProviderChannelStatusProvider{}
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()
	settler := NewHermesPromiseSettler(&mockTransactor{}, channelProvider, channelStatusProvider, mrsp, ks, &settlementHistoryStorageMock{}, cfg)

	// no receive on unknown provider
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, mockProviderChannel, HermesPromise{})
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// no receive should be gotten on a non registered provider
	settler.currentState[mockID] = settlementState{
		registered: false,
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, mockProviderChannel, HermesPromise{})
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// should receive on registered provider. Should also expect a recalculated balance to be added to the settlementState
	expectedChannel := client.ProviderChannel{Balance: big.NewInt(10000), Stake: big.NewInt(1000)}
	expectedPromise := crypto.Promise{Amount: big.NewInt(9000)}
	settler.currentState[mockID] = settlementState{
		registered: true,
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise})
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})

	p := <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)

	// should not receive here due to balance being large and stake being small
	expectedChannel = client.ProviderChannel{Balance: big.NewInt(10000), Stake: big.NewInt(0)}
	expectedPromise = crypto.Promise{Amount: big.NewInt(8900)}
	settler.currentState[mockID] = settlementState{
		registered:       true,
		settleInProgress: false,
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, mockProviderChannel, HermesPromise{Promise: expectedPromise})
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})
	assertNoReceive(t, settler.settleQueue)
}

func assertNoReceive(t *testing.T, ch chan receivedPromise) {
	// at this point, we should not receive an event on settled queue as we have no info on provider, let's check for that
	select {
	case <-ch:
		assert.Fail(t, "did not expect to receive from settled queue")
	case <-time.After(time.Millisecond * 2):
		// we've not received on the channel, continue
	}
}

func TestPromiseSettler_handleNodeStart(t *testing.T) {
	ks := identity.NewMockKeystore()

	acc1, err := ks.NewAccount("")
	assert.NoError(t, err)
	acc2, err := ks.NewAccount("")
	assert.NoError(t, err)

	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			identity.FromAddress(acc2.Address.Hex()): {
				status: registry.Registered,
			},
			identity.FromAddress(acc1.Address.Hex()): {
				status: registry.Unregistered,
			},
		},
	}

	settler := NewHermesPromiseSettler(&mockTransactor{}, &mockHermesChannelProvider{}, &mockProviderChannelStatusProvider{}, mrsp, ks, &settlementHistoryStorageMock{}, cfg)

	settler.handleNodeStart()

	// since each address is checked on BC in background, we'll need to wait here until that is complete
	time.Sleep(time.Millisecond * 10)

	// since we're accessing the current state from outside the setller, lock the settler to prevent race conditions
	settler.lock.Lock()
	defer settler.lock.Unlock()

	assert.True(t, settler.currentState[identity.FromAddress(acc2.Address.Hex())].registered)
	assert.False(t, settler.currentState[identity.FromAddress(acc1.Address.Hex())].registered)
}

func TestPromiseSettlerState_needsSettling(t *testing.T) {
	s := settlementState{
		registered: true,
	}
	channel := NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Balance: big.NewInt(100), Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(100)}},
	)
	assert.True(t, s.needsSettling(0.1, channel), "should be true with zero balance left")

	s = settlementState{
		registered: true,
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Balance: big.NewInt(10000), Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(9000)}},
	)
	assert.True(t, s.needsSettling(0.1, channel), "should be true with 10% missing")

	s.registered = false
	assert.False(t, s.needsSettling(0.1, channel), "should be false with no registration")

	s.settleInProgress = true
	assert.False(t, s.needsSettling(0.1, channel), "should be false with settle in progress")

	s = settlementState{
		registered: true,
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Balance: big.NewInt(10000), Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(8999)}},
	)
	assert.False(t, s.needsSettling(0.1, channel), "should be false with 10.01% missing")
}

// mocks start here
type mockProviderChannelStatusProvider struct {
	channelToReturn    client.ProviderChannel
	channelReturnError error
	sinkToReturn       chan *bindings.HermesImplementationPromiseSettled
	subCancel          func()
	subError           error
	feeToReturn        uint16
	feeError           error
}

func (mpcsp *mockProviderChannelStatusProvider) SubscribeToPromiseSettledEvent(providerID, hermesID common.Address) (sink chan *bindings.HermesImplementationPromiseSettled, cancel func(), err error) {
	return mpcsp.sinkToReturn, mpcsp.subCancel, mpcsp.subError
}

func (mpcsp *mockProviderChannelStatusProvider) GetProviderChannel(hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error) {
	return mpcsp.channelToReturn, mpcsp.channelReturnError
}

func (mpcsp *mockProviderChannelStatusProvider) GetHermesFee(hermesAddress common.Address) (uint16, error) {
	return mpcsp.feeToReturn, mpcsp.feeError
}

var cfg = HermesPromiseSettlerConfig{
	HermesAddress:        common.HexToAddress("0x9a8B6d979e188fA3DeAa93A470C3537362FdaE92"),
	Threshold:            0.1,
	MaxWaitForSettlement: time.Millisecond * 10,
}

type mockHermesChannelProvider struct {
	channelToReturn    HermesChannel
	channelReturnError error
}

func (mhcp *mockHermesChannelProvider) Get(_ identity.Identity, _ common.Address) (HermesChannel, bool) {
	return mhcp.channelToReturn, true
}

func (mhcp *mockHermesChannelProvider) Fetch(_ identity.Identity, _ common.Address) (HermesChannel, error) {
	return mhcp.channelToReturn, mhcp.channelReturnError
}

type mockRegistrationStatus struct {
	status registry.RegistrationStatus
	err    error
}

type mockRegistrationStatusProvider struct {
	identities map[identity.Identity]mockRegistrationStatus
}

func (mrsp *mockRegistrationStatusProvider) GetRegistrationStatus(id identity.Identity) (registry.RegistrationStatus, error) {
	if v, ok := mrsp.identities[id]; ok {
		return v.status, v.err
	}

	return registry.Unregistered, nil
}

var errMock = errors.New("explosions everywhere")
var mockID = identity.FromAddress("0x0000000000000000000000000000000000000001")
var hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")

var mockProviderChannel = client.ProviderChannel{
	Balance: big.NewInt(1000000000000),
	Settled: big.NewInt(9000000),
	Stake:   big.NewInt(12312323),
}

type mockTransactor struct {
	feesError    error
	feesToReturn registry.FeesResponse

	statusToReturn registry.TransactorStatusResponse
	statusError    error
}

func (mt *mockTransactor) FetchSettleFees() (registry.FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

func (mt *mockTransactor) SettleAndRebalance(_, _ string, _ crypto.Promise) error {
	return nil
}

func (mt *mockTransactor) SettleWithBeneficiary(_, _, _ string, _ crypto.Promise) error {
	return nil
}

func (mt *mockTransactor) SettleIntoStake(accountantID, providerID string, promise crypto.Promise) error {
	return nil
}

func (mt *mockTransactor) FetchRegistrationStatus(id string) (registry.TransactorStatusResponse, error) {
	return mt.statusToReturn, mt.statusError
}

type settlementHistoryStorageMock struct{}

func (shsm *settlementHistoryStorageMock) Store(_ SettlementHistoryEntry) error {
	return nil
}
