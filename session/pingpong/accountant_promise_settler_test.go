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
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestPromiseSettler_resyncState_returns_errors(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelReturnError: errMock,
	}
	mrsp := &mockRegistrationStatusProvider{}
	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "testPromiseSettler_resyncState_returns_errors")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)
	err = settler.resyncState(mockID)
	assert.Equal(t, fmt.Sprintf("could not get provider channel for %v: %v", mockID, errMock.Error()), err.Error())

	channelStatusProvider.channelReturnError = nil
	mapg.err = errMock
	err = settler.resyncState(mockID)
	assert.Equal(t, fmt.Sprintf("could not get accountant promise for %v: %v", mockID, errMock.Error()), err.Error())
}

func TestPromiseSettler_resyncState_handles_no_promise(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{}
	mapg := &mockAccountantPromiseGetter{
		err: ErrNotFound,
	}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_resyncState_handles_no_promise")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	id := identity.FromAddress("test")
	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)
	err = settler.resyncState(id)
	assert.NoError(t, err)

	v := settler.currentState[id]
	expectedBalance := channelStatusProvider.channelToReturn.Balance.Uint64() + channelStatusProvider.channelToReturn.Settled.Uint64()
	assert.Equal(t, expectedBalance, v.balance())
	assert.Equal(t, expectedBalance, v.availableBalance())
	assert.True(t, v.registered)
}

func TestPromiseSettler_resyncState_takes_promise_into_account(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{}
	mapg := &mockAccountantPromiseGetter{
		promise: AccountantPromise{
			Promise: crypto.Promise{
				Amount: 7000000,
			},
		},
	}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_resyncState_takes_promise_into_account")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)
	err = settler.resyncState(mockID)
	assert.NoError(t, err)

	v := settler.currentState[mockID]
	expectedBalance := channelStatusProvider.channelToReturn.Balance.Uint64() + channelStatusProvider.channelToReturn.Settled.Uint64() - mapg.promise.Promise.Amount
	assert.Equal(t, expectedBalance, v.balance())
	assert.Equal(t, channelStatusProvider.channelToReturn.Balance.Uint64()+channelStatusProvider.channelToReturn.Settled.Uint64(), v.availableBalance())
	assert.True(t, v.registered)
}

func TestPromiseSettler_loadInitialState(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: mockRegistrationStatus{
				status: registry.RegisteredProvider,
			},
		},
	}
	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_loadInitialState")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)

	settler.currentState[mockID] = settlementState{}

	// check if existing gets skipped
	err = settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v := settler.currentState[mockID]
	assert.EqualValues(t, settlementState{}, v)

	// check if unregistered gets skipped
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.RegisteredConsumer,
	}

	err = settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	assert.EqualValues(t, settlementState{}, v)

	// check if will resync
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.RegisteredProvider,
	}

	err = settler.loadInitialState(mockID)
	assert.NoError(t, err)

	v = settler.currentState[mockID]
	expectedBalance := channelStatusProvider.channelToReturn.Balance.Uint64() + channelStatusProvider.channelToReturn.Settled.Uint64() - mapg.promise.Promise.Amount
	assert.Equal(t, expectedBalance, v.balance())
	assert.Equal(t, channelStatusProvider.channelToReturn.Balance.Uint64()+channelStatusProvider.channelToReturn.Settled.Uint64(), v.availableBalance())
	assert.True(t, v.registered)

	// check if will bubble registration status errors
	delete(settler.currentState, mockID)

	mrsp.identities[mockID] = mockRegistrationStatus{
		status: registry.RegisteredProvider,
		err:    errMock,
	}

	err = settler.loadInitialState(mockID)
	assert.Equal(t, fmt.Sprintf("could not check registration status for %v: %v", mockID, errMock.Error()), err.Error())
}

func TestPromiseSettler_handleServiceEvent(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: mockRegistrationStatus{
				status: registry.RegisteredProvider,
			},
		},
	}
	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_handleServiceEvent")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)

	statusesWithNoChangeExpected := []string{string(servicestate.Starting), string(servicestate.NotRunning)}

	for _, v := range statusesWithNoChangeExpected {
		settler.handleServiceEvent(servicestate.AppEventServiceStatus{
			ProviderID: mockID.Address,
			Status:     v,
		})

		_, ok := settler.currentState[mockID]

		assert.False(t, ok)
	}

	settler.handleServiceEvent(servicestate.AppEventServiceStatus{
		ProviderID: mockID.Address,
		Status:     string(servicestate.Running),
	})

	_, ok := settler.currentState[mockID]
	assert.True(t, ok)
}

func TestPromiseSettler_handleRegistrationEvent(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: mockRegistrationStatus{
				status: registry.RegisteredProvider,
			},
		},
	}
	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_handleRegistrationEvent")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)

	statusesWithNoChangeExpected := []registry.RegistrationStatus{registry.RegisteredConsumer, registry.Unregistered, registry.InProgress, registry.Promoting, registry.RegistrationError}
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
		Status: registry.RegisteredProvider,
	})

	_, ok := settler.currentState[mockID]
	assert.True(t, ok)
}

func TestPromiseSettler_handleAccountantPromiseReceived(t *testing.T) {
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			mockID: mockRegistrationStatus{
				status: registry.RegisteredProvider,
			},
		},
	}
	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_handleAccountantPromiseReceived")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	// no receive on unknown provider
	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)
	settler.handleAccountantPromiseReceived(event.AppEventAccountantPromise{
		AccountantID: cfg.AccountantAddress,
		ProviderID:   mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// no receive should be gotten on a non registered provider
	settler.currentState[mockID] = settlementState{
		registered: false,
	}
	settler.handleAccountantPromiseReceived(event.AppEventAccountantPromise{
		AccountantID: cfg.AccountantAddress,
		ProviderID:   mockID,
	})
	assertNoReceive(t, settler.settleQueue)

	// should receive on registered provider. Should also expect a recalculated balance to be added to the settlementState
	settler.currentState[mockID] = settlementState{
		channel:     client.ProviderChannel{Balance: big.NewInt(10000)},
		lastPromise: crypto.Promise{Amount: 8900},
		registered:  true,
	}
	settler.handleAccountantPromiseReceived(event.AppEventAccountantPromise{
		AccountantID: cfg.AccountantAddress,
		ProviderID:   mockID,
		Promise:      crypto.Promise{Amount: 9000},
	})

	p := <-settler.settleQueue
	assert.Equal(t, mockID, p.provider)

	v := settler.currentState[mockID]
	assert.Equal(t, uint64(10000-9000), v.balance())

	// should not receive here due to balance being large and stake being small
	settler.currentState[mockID] = settlementState{
		channel:     client.ProviderChannel{Balance: big.NewInt(10000)},
		lastPromise: crypto.Promise{Amount: 8900},
		registered:  true,
	}
	settler.handleAccountantPromiseReceived(event.AppEventAccountantPromise{
		AccountantID: cfg.AccountantAddress,
		ProviderID:   mockID,
		Promise: crypto.Promise{
			Amount: 8999,
		},
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
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}

	mapg := &mockAccountantPromiseGetter{}
	dir, err := ioutil.TempDir("", "TestPromiseSettler_handleAccountantPromiseReceived")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)

	acc1, err := ks.NewAccount("")
	assert.NoError(t, err)
	acc2, err := ks.NewAccount("")
	assert.NoError(t, err)

	mrsp := &mockRegistrationStatusProvider{
		identities: map[identity.Identity]mockRegistrationStatus{
			identity.FromAddress(acc2.Address.Hex()): mockRegistrationStatus{
				status: registry.RegisteredProvider,
			},
			identity.FromAddress(acc1.Address.Hex()): mockRegistrationStatus{
				status: registry.RegisteredConsumer,
			},
		},
	}

	settler := NewAccountantPromiseSettler(eventbus.New(), &mockTransactor{}, mapg, channelStatusProvider, mrsp, ks, cfg)

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
		channel:     client.ProviderChannel{Balance: big.NewInt(100)},
		lastPromise: crypto.Promise{Amount: 100},
		registered:  true,
	}
	assert.True(t, s.needsSettling(0.1), "should be true with zero balance left")

	s = settlementState{
		channel:     client.ProviderChannel{Balance: big.NewInt(10000)},
		lastPromise: crypto.Promise{Amount: 9000},
		registered:  true,
	}
	assert.True(t, s.needsSettling(0.1), "should be true with 10% missing")

	s.registered = false
	assert.False(t, s.needsSettling(0.1), "should be false with no registration")

	s.settleInProgress = true
	assert.False(t, s.needsSettling(0.1), "should be false with settle in progress")

	s = settlementState{
		channel:     client.ProviderChannel{Balance: big.NewInt(10000)},
		lastPromise: crypto.Promise{Amount: 8999},
		registered:  true,
	}
	assert.False(t, s.needsSettling(0.1), "should be false with 10.01% missing")
}

func TestPromiseSettlerState_balance(t *testing.T) {
	s := settlementState{
		channel: client.ProviderChannel{
			Balance: big.NewInt(100),
			Settled: big.NewInt(10),
		},
		lastPromise: crypto.Promise{
			Amount: 15,
		},
	}
	assert.Equal(t, uint64(110), s.availableBalance())
	assert.Equal(t, uint64(95), s.balance())
	assert.Equal(t, uint64(5), s.unsettledBalance())

	s = settlementState{
		channel: client.ProviderChannel{
			Balance: big.NewInt(100),
			Settled: big.NewInt(10),
		},
		lastPromise: crypto.Promise{
			Amount: 16,
		},
	}
	assert.Equal(t, uint64(110), s.availableBalance())
	assert.Equal(t, uint64(94), s.balance())
	assert.Equal(t, uint64(6), s.unsettledBalance())
}

// mocks start here
type mockProviderChannelStatusProvider struct {
	channelToReturn    client.ProviderChannel
	channelReturnError error
	sinkToReturn       chan *bindings.AccountantImplementationPromiseSettled
	subCancel          func()
	subError           error
}

func (mpcsp *mockProviderChannelStatusProvider) SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error) {
	return mpcsp.sinkToReturn, mpcsp.subCancel, mpcsp.subError
}

func (mpcsp *mockProviderChannelStatusProvider) GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address) (client.ProviderChannel, error) {
	return mpcsp.channelToReturn, mpcsp.channelReturnError
}

var cfg = AccountantPromiseSettlerConfig{
	AccountantAddress:    common.HexToAddress("0x9a8B6d979e188fA3DeAa93A470C3537362FdaE92"),
	Threshold:            0.1,
	MaxWaitForSettlement: time.Millisecond * 10,
}

type mockAccountantPromiseGetter struct {
	promise AccountantPromise
	err     error
}

func (mapg *mockAccountantPromiseGetter) Get(_ identity.Identity, _ common.Address) (AccountantPromise, error) {
	return mapg.promise, mapg.err
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
var mockID = identity.FromAddress("test")

var mockProviderChannel = client.ProviderChannel{
	Balance: big.NewInt(1000000000000),
	Settled: big.NewInt(9000000),
	Loan:    big.NewInt(12312323),
}

type mockTransactor struct {
	registerError error
	feesToReturn  registry.FeesResponse
	feesError     error
}

func (mt *mockTransactor) FetchSettleFees() (registry.FeesResponse, error) {
	return mt.feesToReturn, mt.feesError
}

func (mt *mockTransactor) SettleAndRebalance(id string, promise crypto.Promise) error {
	return nil
}
