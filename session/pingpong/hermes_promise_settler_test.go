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
	"strings"
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
		identities: map[string]mockRegistrationStatus{
			mockChainIdentity: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()

	fac := &mockHermesCallerFactory{}

	settler := NewHermesPromiseSettler(
		&mockTransactor{},
		&mockHermesPromiseStorage{},
		&mockPayAndSettler{},
		&mockAddressProvider{},
		fac.Get,
		&mockHermesURLGetter{},
		&mockHermesChannelProvider{},
		&mockProviderChannelStatusProvider{},
		mrsp,
		ks,
		&settlementHistoryStorageMock{},
		cfg)

	settler.currentState[mockID] = settlementState{}

	// check if existing gets skipped
	err := settler.loadInitialState(0, mockID)
	assert.NoError(t, err)

	v := settler.currentState[mockID]
	assert.EqualValues(t, settlementState{}, v)

	// check if unregistered gets skipped
	delete(settler.currentState, mockID)

	mrsp.identities[mockChainIdentity] = mockRegistrationStatus{
		status: registry.Registered,
	}

	err = settler.loadInitialState(0, mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	assert.EqualValues(t, settlementState{
		registered: true,
	}, v)

	// check if will resync
	delete(settler.currentState, mockID)

	mrsp.identities[mockChainIdentity] = mockRegistrationStatus{
		status: registry.Registered,
	}

	err = settler.loadInitialState(0, mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	assert.True(t, v.registered)

	// check if will bubble registration status errors
	delete(settler.currentState, mockID)

	mrsp.identities[mockChainIdentity] = mockRegistrationStatus{
		status: registry.Registered,
		err:    errMock,
	}

	err = settler.loadInitialState(0, mockID)
	assert.Equal(t, fmt.Sprintf("could not check registration status for %v: %v", mockID, errMock.Error()), err.Error())
}

func TestPromiseSettler_handleRegistrationEvent(t *testing.T) {
	mrsp := &mockRegistrationStatusProvider{
		identities: map[string]mockRegistrationStatus{
			mockChainIdentity: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()
	fac := &mockHermesCallerFactory{}

	settler := NewHermesPromiseSettler(
		&mockTransactor{},
		&mockHermesPromiseStorage{},
		&mockPayAndSettler{},
		&mockAddressProvider{},
		fac.Get,
		&mockHermesURLGetter{},
		&mockHermesChannelProvider{},
		&mockProviderChannelStatusProvider{},
		mrsp,
		ks,
		&settlementHistoryStorageMock{},
		cfg)

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
		identities: map[string]mockRegistrationStatus{
			mockChainIdentity: {
				status: registry.Registered,
			},
		},
	}
	ks := identity.NewMockKeystore()
	fac := &mockHermesCallerFactory{}
	settler := NewHermesPromiseSettler(&mockTransactor{}, &mockHermesPromiseStorage{}, &mockPayAndSettler{},&mockAddressProvider{},fac.Get, &mockHermesURLGetter{}, channelProvider, channelStatusProvider, mrsp, ks, &settlementHistoryStorageMock{}, cfg)

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
	expectedChannel := client.ProviderChannel{Stake: big.NewInt(1000)}
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
	expectedChannel = client.ProviderChannel{Stake: big.NewInt(0)}
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
		identities: map[string]mockRegistrationStatus{
			"0" + strings.ToLower(acc2.Address.Hex()): {
				status: registry.Registered,
			},
			"0" + strings.ToLower(acc1.Address.Hex()): {
				status: registry.Unregistered,
			},
		},
	}

	fac := &mockHermesCallerFactory{}
	settler := NewHermesPromiseSettler(
		&mockTransactor{},
		&mockHermesPromiseStorage{},
		&mockPayAndSettler{},
		&mockAddressProvider{},
		fac.Get,
		&mockHermesURLGetter{},
		&mockHermesChannelProvider{},
		&mockProviderChannelStatusProvider{},
		mrsp,
		ks,
		&settlementHistoryStorageMock{},
		cfg)

	settler.handleNodeStart()

	// since each address is checked on BC in background, we'll need to wait here until that is complete
	time.Sleep(time.Millisecond * 10)

	// since we're accessing the current state from outside the setller, lock the settler to prevent race conditions
	settler.lock.Lock()
	defer settler.lock.Unlock()

	assert.True(t, settler.currentState[identity.FromAddress(acc2.Address.Hex())].registered)
	assert.False(t, settler.currentState[identity.FromAddress(acc1.Address.Hex())].registered)
}

func TestPromiseSettler_RejectsIfFeesExceedSettlementAmount(t *testing.T) {
	fac := &mockHermesCallerFactory{}
	transactorFee := big.NewInt(5000)
	hermesFee := big.NewInt(25000)

	promiseSettler := hermesPromiseSettler{
		currentState: make(map[identity.Identity]settlementState),
		transactor: &mockTransactor{
			feesToReturn: registry.FeesResponse{
				Fee: transactorFee,
			},
		},
		hermesCallerFactory: fac.Get,
		hermesURLGetter:     &mockHermesURLGetter{},
		bc: &mockProviderChannelStatusProvider{
			calculatedFees: hermesFee,
		},
	}

	mockPromise := crypto.Promise{
		Fee:    transactorFee,
		Amount: big.NewInt(35000),
	}

	settled := big.NewInt(6000)

	mockSettler := func(crypto.Promise) error { return nil }
	err := promiseSettler.settle(mockSettler, identity.Identity{}, common.Address{}, mockPromise, common.Address{}, settled)
	assert.Equal(t, "Settlement fees exceed earning amount. Please provide more service and try again. Current earnings: 29000, current fees: 30000", err.Error())
}

func TestPromiseSettler_AcceptsIfFeesDoNotExceedSettlementAmount(t *testing.T) {
	fac := &mockHermesCallerFactory{}
	transactorFee := big.NewInt(5000)
	hermesFee := big.NewInt(20000)
	bc := &mockProviderChannelStatusProvider{
		calculatedFees: hermesFee,
		sinkToReturn:   make(chan *bindings.HermesImplementationPromiseSettled),
		subCancel:      func() {},
	}
	promiseSettler := hermesPromiseSettler{
		currentState: make(map[identity.Identity]settlementState),
		transactor: &mockTransactor{
			feesToReturn: registry.FeesResponse{
				Fee: transactorFee,
			},
		},
		hermesCallerFactory: fac.Get,
		hermesURLGetter:     &mockHermesURLGetter{},
		bc:                  bc,
		channelProvider:     &mockHermesChannelProvider{},
		config: HermesPromiseSettlerConfig{
			MaxWaitForSettlement: time.Millisecond * 50,
		},
		settlementHistoryStorage: &settlementHistoryStorageMock{},
	}

	mockPromise := crypto.Promise{
		Fee:    transactorFee,
		Amount: big.NewInt(35000),
	}

	settled := big.NewInt(6000)

	mockSettler := func(crypto.Promise) error { return nil }

	go func() { bc.sinkToReturn <- &bindings.HermesImplementationPromiseSettled{} }()
	err := promiseSettler.settle(mockSettler, identity.Identity{}, common.Address{}, mockPromise, common.Address{}, settled)
	assert.NoError(t, err)
}

func TestPromiseSettlerState_needsSettling(t *testing.T) {
	s := settlementState{
		registered: true,
	}
	channel := NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(1000)}},
	)
	assert.True(t, s.needsSettling(0.1, channel), "should be true with zero balance left")

	s = settlementState{
		registered: true,
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(1000)},
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
		client.ProviderChannel{Stake: big.NewInt(10000)},
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
	calculatedFees     *big.Int
	calculationError   error
	balanceToReturn    *big.Int
}

func (mpcsp *mockProviderChannelStatusProvider) SubscribeToPromiseSettledEvent(chainID int64, providerID, hermesID common.Address) (sink chan *bindings.HermesImplementationPromiseSettled, cancel func(), err error) {
	return mpcsp.sinkToReturn, mpcsp.subCancel, mpcsp.subError
}

func (mpcsp *mockProviderChannelStatusProvider) GetProviderChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error) {
	return mpcsp.channelToReturn, mpcsp.channelReturnError
}

func (mpcsp *mockProviderChannelStatusProvider) GetHermesFee(chainID int64, hermesAddress common.Address) (uint16, error) {
	return mpcsp.feeToReturn, mpcsp.feeError
}

func (mpcsp *mockProviderChannelStatusProvider) GetBeneficiary(chainID int64, registryAddress, identity common.Address) (common.Address, error) {
	return common.Address{}, nil
}

func (mpcsp *mockProviderChannelStatusProvider) CalculateHermesFee(chainID int64, hermesAddress common.Address, value *big.Int) (*big.Int, error) {
	return mpcsp.calculatedFees, mpcsp.calculationError
}

func (mpcsp *mockProviderChannelStatusProvider) GetMystBalance(chainID int64, mystAddress common.Address, address common.Address) (*big.Int, error) {
	return mpcsp.balanceToReturn, nil
}

var cfg = HermesPromiseSettlerConfig{
	Threshold:            0.1,
	MaxWaitForSettlement: time.Millisecond * 10,
}

type mockHermesChannelProvider struct {
	channelToReturn    HermesChannel
	channelReturnError error
}

func (mhcp *mockHermesChannelProvider) Get(chainID int64, _ identity.Identity, _ common.Address) (HermesChannel, bool) {
	return mhcp.channelToReturn, true
}

func (mhcp *mockHermesChannelProvider) Fetch(chainID int64, _ identity.Identity, _ common.Address) (HermesChannel, error) {
	return mhcp.channelToReturn, mhcp.channelReturnError
}

type mockRegistrationStatus struct {
	status registry.RegistrationStatus
	err    error
}

type mockRegistrationStatusProvider struct {
	identities map[string]mockRegistrationStatus
}

func (mrsp *mockRegistrationStatusProvider) GetRegistrationStatus(chainID int64, id identity.Identity) (registry.RegistrationStatus, error) {
	if v, ok := mrsp.identities[fmt.Sprintf("%d%s", chainID, id.Address)]; ok {
		return v.status, v.err
	}

	return registry.Unregistered, nil
}

var errMock = errors.New("explosions everywhere")
var mockID = identity.FromAddress("0x0000000000000000000000000000000000000001")
var hermesID = common.HexToAddress("0x00000000000000000000000000000000000000002")
var mockChainIdentity = "0" + mockID.Address

var mockProviderChannel = client.ProviderChannel{
	Settled: big.NewInt(9000000),
	Stake:   big.NewInt(1000000000000),
}

type mockTransactor struct {
	feesError    error
	feesToReturn registry.FeesResponse

	statusToReturn registry.TransactorStatusResponse
	statusError    error
}

func (mt *mockTransactor) FetchSettleFees(chainID int64) (registry.FeesResponse, error) {
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

func (mt *mockTransactor) PayAndSettle(hermesID, providerID string, promise crypto.Promise, beneficiary string, beneficiarySignature string) error {
	return nil
}

func (mt *mockTransactor) FetchRegistrationStatus(id string) ([]registry.TransactorStatusResponse, error) {
	return []registry.TransactorStatusResponse{mt.statusToReturn}, mt.statusError
}

func (mt *mockTransactor) FetchRegistrationFees(chainID int64) (registry.FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

type settlementHistoryStorageMock struct{}

func (shsm *settlementHistoryStorageMock) Store(_ SettlementHistoryEntry) error {
	return nil
}

type mockPayAndSettler struct{}

func (mpas *mockPayAndSettler) PayAndSettle(r []byte, em crypto.ExchangeMessage, providerID identity.Identity, sessionID string) <-chan error {
	return nil
}
