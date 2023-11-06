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
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/mysteriumnetwork/payments/units"
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
	lbs := newMockAddressStorage()

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
		&mockPublisher{},
		&mockObserver{},
		lbs,
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
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
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
	lbs := newMockAddressStorage()

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
		&mockPublisher{},
		&mockObserver{},
		lbs,
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
	tm := &mockTransactor{
		feesToReturn: registry.FeesResponse{
			Fee:        units.FloatEthToBigIntWei(0.05),
			ValidUntil: time.Now().Add(30 * time.Minute),
		},
	}
	lbs := newMockAddressStorage()

	settler := NewHermesPromiseSettler(tm, &mockHermesPromiseStorage{}, &mockPayAndSettler{}, &mockAddressProvider{}, fac.Get, &mockHermesURLGetter{}, channelProvider, channelStatusProvider, mrsp, ks, &settlementHistoryStorageMock{}, &mockPublisher{}, &mockObserver{}, lbs, cfg)

	// no receive on unknown provider
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, mockProviderChannel, HermesPromise{}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// no receive should be gotten on a non registered provider
	settler.currentState[mockID] = settlementState{
		registered:       false,
		settleInProgress: map[common.Address]struct{}{},
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, mockProviderChannel, HermesPromise{}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// should receive on registered provider. Should also expect a recalculated balance to be added to the settlementState
	expectedChannel := client.ProviderChannel{Stake: big.NewInt(1000)}
	expectedPromise := crypto.Promise{Amount: units.FloatEthToBigIntWei(6)}
	settler.currentState[mockID] = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})

	p := <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)

	// should not receive here due to balance being large and stake being small
	expectedChannel = client.ProviderChannel{
		Stake:   big.NewInt(0),
		Settled: units.FloatEthToBigIntWei(6),
	}
	expectedPromise = crypto.Promise{Amount: units.FloatEthToBigIntWei(8)}
	settler.currentState[mockID] = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})
	assertNoReceive(t, settler.settleQueue)
}

func TestPromiseSettler_handleHermesPromiseReceivedWithLocalBeneficiary(t *testing.T) {
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
	tm := &mockTransactor{
		feesToReturn: registry.FeesResponse{
			Fee:        units.FloatEthToBigIntWei(0.05),
			ValidUntil: time.Now().Add(30 * time.Minute),
		},
	}
	lbs := newMockAddressStorage()

	settler := NewHermesPromiseSettler(tm, &mockHermesPromiseStorage{}, &mockPayAndSettler{}, &mockAddressProvider{}, fac.Get, &mockHermesURLGetter{}, channelProvider, channelStatusProvider, mrsp, ks, &settlementHistoryStorageMock{}, &mockPublisher{}, &mockObserver{}, lbs, cfg)

	expectedChannel := client.ProviderChannel{Stake: big.NewInt(1000)}
	expectedPromise := crypto.Promise{Amount: units.FloatEthToBigIntWei(6)}
	settler.currentState[mockID] = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})

	p := <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)
	assert.Equal(t, beneficiaryID, p.beneficiary)

	localBeneficiary := common.HexToAddress("0x00000000000000000000000000000000000000133")
	err := lbs.Save(mockID.Address, localBeneficiary.Hex())
	assert.NoError(t, err)

	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})
	p = <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)
}

func TestPromiseSettler_doNotSettleIfBeneficiaryIsAChannel(t *testing.T) {
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
	tm := &mockTransactor{
		feesToReturn: registry.FeesResponse{
			Fee:        units.FloatEthToBigIntWei(0.05),
			ValidUntil: time.Now().Add(30 * time.Minute),
		},
	}
	lbs := newMockAddressStorage()

	mockAddressProvider := newMockAddressProvider()
	settler := NewHermesPromiseSettler(tm, &mockHermesPromiseStorage{}, &mockPayAndSettler{}, mockAddressProvider, fac.Get, &mockHermesURLGetter{}, channelProvider, channelStatusProvider, mrsp, ks, &settlementHistoryStorageMock{}, &mockPublisher{}, &mockObserver{}, lbs, cfg)

	expectedChannel := client.ProviderChannel{Stake: big.NewInt(1000)}
	expectedPromise := crypto.Promise{Amount: units.FloatEthToBigIntWei(6)}
	settler.currentState[mockID] = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})

	p := <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)
	assert.Equal(t, beneficiaryID, p.beneficiary)

	mockAddressProvider.setChannelAddress(0, mockID.ToCommonAddress(), beneficiaryID)
	channelProvider.channelToReturn = NewHermesChannel("1", mockID, hermesID, expectedChannel, HermesPromise{Promise: expectedPromise}, beneficiaryID)
	settler.handleHermesPromiseReceived(event.AppEventHermesPromise{
		HermesID:   hermesID,
		ProviderID: mockID,
		Promise:    expectedPromise,
	})

	p = <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)
	assert.Equal(t, beneficiaryID, p.beneficiary)
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

	lbs := newMockAddressStorage()

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
		&mockPublisher{},
		&mockObserver{},
		lbs,
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
		currentState: map[identity.Identity]settlementState{},
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

	mockSettler := func(crypto.Promise) (string, error) { return "", nil }
	err := promiseSettler.settle(mockSettler, identity.Identity{}, common.Address{}, mockPromise, common.Address{}, settled, nil)
	assert.Equal(t, "settlement fees exceed earning amount. Please provide more service and try again. Current earnings: 29000, current fees: 30000: fee not covered, cannot continue", err.Error())
}

func TestPromiseSettler_RejectsIfFeesExceedMaxFee(t *testing.T) {
	fac := &mockHermesCallerFactory{}
	transactorFee := units.FloatEthToBigIntWei(0.8)
	hermesFee := big.NewInt(25000)

	promiseSettler := hermesPromiseSettler{
		currentState: map[identity.Identity]settlementState{},
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

	mockSettler := func(crypto.Promise) (string, error) { return "", nil }
	err := promiseSettler.settle(mockSettler, identity.Identity{}, common.Address{}, mockPromise, common.Address{}, settled, units.FloatEthToBigIntWei(0.6))
	assert.Equal(t, "current fee is more than the max", err.Error())
}

func TestPromiseSettler_AcceptsIfFeesDoNotExceedSettlementAmount(t *testing.T) {
	fac := &mockHermesCallerFactory{}
	transactorFee := big.NewInt(5000)
	hermesFee := big.NewInt(20000)
	expectedChannel, err := hex.DecodeString("d0bb35eb0e4a0c972f2c154f91cf676b804762bef69c7fe4cef38642c3ac7ffc")
	assert.NoError(t, err)

	expectedR, err := hex.DecodeString("d56e23228dc2c7d2cc2e0ee08d7d6e5be6aa196c9f95046d83fab06913d2a9c2")
	assert.NoError(t, err)

	var arr [32]byte
	copy(arr[:], expectedChannel)

	var r [32]byte
	copy(r[:], expectedR)
	bc := &mockProviderChannelStatusProvider{
		calculatedFees: hermesFee,
		subCancel:      func() {},
		promiseEventsToReturn: []bindings.HermesImplementationPromiseSettled{
			{
				ChannelId: arr,
				Lock:      r,
			},
		},
		headerToReturn: &types.Header{
			Number: big.NewInt(0),
		},
	}

	publisher := &mockPublisher{
		publicationChan: make(chan testEvent, 10),
	}
	promiseSettler := hermesPromiseSettler{
		currentState: make(map[identity.Identity]settlementState),
		transactor: &mockTransactor{
			idToReturn: "123",
			queueToReturn: registry.QueueResponse{
				State: "done",
			},
			feesToReturn: registry.FeesResponse{
				Fee: transactorFee,
			},
		},
		hermesCallerFactory: fac.Get,
		hermesURLGetter:     &mockHermesURLGetter{},
		bc:                  bc,
		channelProvider:     &mockHermesChannelProvider{},
		config: HermesPromiseSettlerConfig{
			SettlementCheckTimeout: time.Millisecond * 50,
		},
		settlementHistoryStorage: &settlementHistoryStorageMock{},
		publisher:                publisher,
	}

	mockPromise := crypto.Promise{
		Fee:    transactorFee,
		Amount: big.NewInt(35000),
		R:      r[:],
	}

	settled := big.NewInt(6000)

	mockSettler := func(crypto.Promise) (string, error) { return "", nil }

	err = promiseSettler.settle(mockSettler, identity.Identity{Address: "0x92fE1c838b08dB4c072DDa805FB4292d9b76B5E7"}, common.HexToAddress("0x07b5fD382b5e375F202184052BeF2C50b3B1404F"), mockPromise, common.Address{}, settled, nil)
	assert.NoError(t, err)
	ev := <-publisher.publicationChan
	assert.Equal(t, event.AppTopicSettlementComplete, ev.name)
	_, ok := ev.value.(event.AppEventSettlementComplete)
	assert.True(t, ok)
}

func TestPromiseSettlerState_needsSettling(t *testing.T) {
	hps := &hermesPromiseSettler{
		transactor: &mockTransactor{
			feesToReturn: registry.FeesResponse{
				Fee:        units.FloatEthToBigIntWei(2.0),
				ValidUntil: time.Now().Add(30 * time.Minute),
			},
		},
	}
	s := settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel := NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(0)},
		HermesPromise{Promise: crypto.Promise{Amount: units.FloatEthToBigIntWei(10.1)}},
		beneficiaryID,
	)
	needs, maxFee := hps.needsSettling(s, 0, 0.1, 5, 10, channel, 1)
	assert.True(t, needs, "should be true with balance more than max regardless of fees")
	assert.Nil(t, maxFee, "should be nil")

	hps = &hermesPromiseSettler{
		transactor: &mockTransactor{
			feesToReturn: registry.FeesResponse{
				Fee:        units.FloatEthToBigIntWei(0.045),
				ValidUntil: time.Now().Add(30 * time.Minute),
			},
		},
	}
	s = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(0)},
		HermesPromise{Promise: crypto.Promise{Amount: units.FloatEthToBigIntWei(5)}},
		beneficiaryID,
	)

	needs, maxFee = hps.needsSettling(s, 0, 0.01, 5, 10, channel, 1)
	assert.True(t, needs, "should be true if fees are 1%% of unsettled amount")
	assert.True(t, maxFee.Cmp(units.FloatEthToBigIntWei(0.045)) > 0, "should be bigger than current fee")

	s.registered = false
	needs, _ = hps.needsSettling(s, 0, 0.01, 5, 10, channel, 1)
	assert.False(t, needs, "should be false with no registration")

	s.settleInProgress = map[common.Address]struct{}{
		hermesID: {},
	}
	needs, _ = hps.needsSettling(s, 0, 0.01, 5, 10, channel, 1)
	assert.False(t, needs, "should be false with settle in progress")

	hps = &hermesPromiseSettler{
		transactor: &mockTransactor{
			feesToReturn: registry.FeesResponse{
				Fee:        units.FloatEthToBigIntWei(0.051),
				ValidUntil: time.Now().Add(30 * time.Minute),
			},
		},
	}
	s = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(0)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(8999)}},
		beneficiaryID,
	)
	needs, _ = hps.needsSettling(s, 0, 0.01, 5, 10, channel, 1)
	assert.False(t, needs, "should be false with fee more than 1%% of unsettled amount")

	s = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(1000)}},
		beneficiaryID,
	)
	needs, maxFee = hps.needsSettling(s, 0.1, 0.01, 5, 10, channel, 1)
	assert.True(t, needs, "should be true with zero balance left")
	assert.Nil(t, maxFee, "should be nil")

	s = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(1000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(9000)}},
		beneficiaryID,
	)
	needs, maxFee = hps.needsSettling(s, 0.1, 0.01, 5, 10, channel, 1)
	assert.True(t, needs, "should be true with 10% missing")
	assert.Nil(t, maxFee, "should be nil")

	s = settlementState{
		registered:       true,
		settleInProgress: map[common.Address]struct{}{},
	}
	channel = NewHermesChannel(
		"1",
		mockID,
		hermesID,
		client.ProviderChannel{Stake: big.NewInt(10000)},
		HermesPromise{Promise: crypto.Promise{Amount: big.NewInt(8999)}},
		beneficiaryID,
	)
	needs, _ = hps.needsSettling(s, 0.1, 0.01, 5, 10, channel, 1)
	assert.False(t, needs, "should be false with 10.01% missing")
}

// mocks start here
type mockProviderChannelStatusProvider struct {
	channelToReturn       client.ProviderChannel
	channelReturnError    error
	sinkToReturn          chan *bindings.HermesImplementationPromiseSettled
	subCancel             func()
	subError              error
	feeToReturn           uint16
	feeError              error
	calculatedFees        *big.Int
	calculationError      error
	balanceToReturn       *big.Int
	headerToReturn        *types.Header
	errorToReturn         error
	promiseEventsToReturn []bindings.HermesImplementationPromiseSettled
	promiseError          error
}

func (mpcsp *mockProviderChannelStatusProvider) TransactionReceipt(chainID int64, hash common.Hash) (*types.Receipt, error) {
	r := &types.Receipt{}
	r.Status = types.ReceiptStatusSuccessful
	return r, nil
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

func (mpcsp *mockProviderChannelStatusProvider) GetProvidersWithdrawalChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error) {
	return client.ProviderChannel{}, nil
}

func (mpcsp *mockProviderChannelStatusProvider) FilterPromiseSettledEventByChannelID(chainID int64, from uint64, to *uint64, hermesID common.Address, providerAddresses [][32]byte) ([]bindings.HermesImplementationPromiseSettled, error) {
	return mpcsp.promiseEventsToReturn, mpcsp.promiseError
}

func (mpcsp *mockProviderChannelStatusProvider) HeaderByNumber(chainID int64, number *big.Int) (*types.Header, error) {
	return mpcsp.headerToReturn, mpcsp.errorToReturn
}

var cfg = HermesPromiseSettlerConfig{
	MaxFeeThreshold:         0.01,
	MinAutoSettleAmount:     5,
	MaxUnSettledAmount:      20,
	SettlementCheckTimeout:  time.Millisecond * 10,
	SettlementCheckInterval: time.Millisecond * 1,
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
var beneficiaryID = common.HexToAddress("0x00000000000000000000000000000000000000132")
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

	queueToReturn registry.QueueResponse
	queueError    error

	idToReturn  string
	settleError error
}

func (mt *mockTransactor) FetchSettleFees(chainID int64) (registry.FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

func (mt *mockTransactor) SettleAndRebalance(_, _ string, _ crypto.Promise) (string, error) {
	return mt.idToReturn, mt.settleError
}

func (mt *mockTransactor) SettleWithBeneficiary(_, _, _ string, _ crypto.Promise) (string, error) {
	return mt.idToReturn, mt.settleError
}

func (mt *mockTransactor) SettleIntoStake(accountantID, providerID string, promise crypto.Promise) (string, error) {
	return mt.idToReturn, mt.settleError
}

func (mt *mockTransactor) PayAndSettle(hermesID, providerID string, promise crypto.Promise, beneficiary string, beneficiarySignature string) (string, error) {
	return mt.idToReturn, mt.settleError
}

func (mt *mockTransactor) FetchRegistrationStatus(id string) ([]registry.TransactorStatusResponse, error) {
	return []registry.TransactorStatusResponse{mt.statusToReturn}, mt.statusError
}

func (mt *mockTransactor) FetchRegistrationFees(chainID int64) (registry.FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

func (mt *mockTransactor) GetQueueStatus(ID string) (registry.QueueResponse, error) {
	return mt.queueToReturn, mt.queueError
}

type settlementHistoryStorageMock struct{}

func (shsm *settlementHistoryStorageMock) Store(_ SettlementHistoryEntry) error {
	return nil
}

type mockPayAndSettler struct{}

func (mpas *mockPayAndSettler) PayAndSettle(r []byte, em crypto.ExchangeMessage, providerID identity.Identity, sessionID string) <-chan error {
	return nil
}

type mockAddressStorage struct {
	data map[string]string
}

func newMockAddressStorage() *mockAddressStorage {
	return &mockAddressStorage{data: make(map[string]string)}
}

func (m mockAddressStorage) Address(identity string) (string, error) {
	return m.data[identity], nil
}

func (m mockAddressStorage) Save(identity, address string) error {
	m.data[identity] = address
	return nil
}
