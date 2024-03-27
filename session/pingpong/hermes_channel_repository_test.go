/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/copier"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHermesChannelRepository_Fetch_returns_errors(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")
	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{}
	mockBeneficiaryProvider := &mockBeneficiaryProvider{}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}

	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus(), mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})

	// when
	channelStatusProvider.channelReturnError = errMock
	promiseProvider.errToReturn = nil
	_, err := repo.Fetch(1, id, hermesID)
	// then
	assert.Errorf(t, err, "could not get provider channel for %v, hermes %v: %v", mockID, common.Address{}.Hex(), errMock.Error())

	// when
	channelStatusProvider.channelReturnError = nil
	promiseProvider.errToReturn = errMock
	_, err = repo.Fetch(1, mockID, hermesID)
	// then
	assert.Errorf(t, err, "could not get hermes promise for provider %v, hermes %v: %v", mockID, common.Address{}.Hex(), errMock.Error())

}

func TestHermesChannelRepository_Fetch_handles_no_promise(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")

	expectedPromise := HermesPromise{}
	promiseProvider := &mockHermesPromiseStorage{
		toReturn:    expectedPromise,
		errToReturn: ErrNotFound,
	}

	expectedChannelStatus := client.ProviderChannel{
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(1000000000000),
	}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: expectedChannelStatus,
	}

	mockBeneficiaryProvider := &mockBeneficiaryProvider{}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}
	// when
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus(), mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})
	channel, err := repo.Fetch(1, id, hermesID)
	assert.NoError(t, err)

	// then
	expectedBalance := new(big.Int).Add(expectedChannelStatus.Stake, expectedChannelStatus.Settled)
	assert.Equal(t, expectedBalance, channel.balance())
	assert.Equal(t, expectedBalance, channel.availableBalance())
}

func TestHermesChannelRepository_Fetch_takes_promise_into_account(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")

	expectedPromise := HermesPromise{
		Promise: crypto.Promise{Amount: big.NewInt(7000000)},
	}
	promiseProvider := &mockHermesPromiseStorage{
		toReturn: expectedPromise,
	}

	expectedChannelStatus := client.ProviderChannel{
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(1000000000000),
	}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: expectedChannelStatus,
	}
	mockBeneficiaryProvider := &mockBeneficiaryProvider{}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}

	// when
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, mocks.NewEventBus(), mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})
	channel, err := repo.Fetch(1, id, hermesID)
	assert.NoError(t, err)

	// then
	added := new(big.Int).Add(expectedChannelStatus.Stake, expectedChannelStatus.Settled)
	expectedBalance := added.Sub(added, expectedPromise.Promise.Amount)
	assert.Equal(t, expectedBalance, channel.balance())
	assert.Equal(t, new(big.Int).Add(expectedChannelStatus.Stake, expectedChannelStatus.Settled), channel.availableBalance())
}

func TestHermesChannelRepository_Fetch_publishesEarningChanges(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")
	expectedPromise1 := HermesPromise{
		ChannelID: "1",
		Promise:   crypto.Promise{Amount: big.NewInt(7000000)},
	}
	expectedPromise2 := HermesPromise{
		ChannelID: "1",
		Promise:   crypto.Promise{Amount: big.NewInt(8000000)},
	}
	expectedChannelStatus1 := client.ProviderChannel{
		Settled: big.NewInt(9000000),
		Stake:   big.NewInt(1000000000000),
	}
	expectedChannelStatus2 := client.ProviderChannel{
		Settled: big.NewInt(9000001),
		Stake:   big.NewInt(1000000000001),
	}

	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{}
	publisher := mocks.NewEventBus()
	mockBeneficiaryProvider := &mockBeneficiaryProvider{
		b: beneficiaryID,
	}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, publisher, mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})

	// when
	promiseProvider.toReturn = expectedPromise1
	channelStatusProvider.channelToReturn = expectedChannelStatus1
	channel, err := repo.Fetch(1, id, hermesID)
	assert.NoError(t, err)

	// then
	expectedChannel1 := NewHermesChannel("1", id, hermesID, expectedChannelStatus1, expectedPromise1, beneficiaryID)
	assert.Equal(t, expectedChannel1, channel)
	assert.Eventually(t, func() bool {
		lastEvent, ok := publisher.Pop().(event.AppEventEarningsChanged)
		if !ok {
			return false
		}
		assert.Equal(
			t,
			event.AppEventEarningsChanged{
				Identity: id,
				Previous: event.EarningsDetailed{
					Total: event.Earnings{
						LifetimeBalance:  big.NewInt(0),
						UnsettledBalance: big.NewInt(0),
					},
					PerHermes: map[common.Address]event.Earnings{},
				},
				Current: event.EarningsDetailed{
					Total: event.Earnings{
						LifetimeBalance:  expectedChannel1.LifetimeBalance(),
						UnsettledBalance: expectedChannel1.UnsettledBalance(),
					},
					PerHermes: map[common.Address]event.Earnings{
						hermesID: {
							LifetimeBalance:  expectedChannel1.LifetimeBalance(),
							UnsettledBalance: expectedChannel1.UnsettledBalance(),
						},
					},
				},
			},
			lastEvent,
		)
		return true
	}, 2*time.Second, 10*time.Millisecond)

	// when
	promiseProvider.toReturn = expectedPromise2
	channelStatusProvider.channelToReturn = expectedChannelStatus2
	channel, err = repo.Fetch(1, id, hermesID)
	assert.NoError(t, err)

	// then
	expectedChannel2 := NewHermesChannel("1", id, hermesID, expectedChannelStatus2, expectedPromise2, beneficiaryID)
	assert.Equal(t, expectedChannel2, channel)
	assert.Eventually(t, func() bool {
		lastEvent, ok := publisher.Pop().(event.AppEventEarningsChanged)
		if !ok {
			return false
		}
		assert.Equal(
			t,
			event.AppEventEarningsChanged{
				Identity: id,
				Previous: event.EarningsDetailed{
					Total: event.Earnings{
						LifetimeBalance:  expectedChannel1.LifetimeBalance(),
						UnsettledBalance: expectedChannel1.UnsettledBalance(),
					},
					PerHermes: map[common.Address]event.Earnings{
						hermesID: {
							LifetimeBalance:  expectedChannel1.LifetimeBalance(),
							UnsettledBalance: expectedChannel1.UnsettledBalance(),
						},
					},
				},
				Current: event.EarningsDetailed{
					Total: event.Earnings{
						LifetimeBalance:  expectedChannel2.LifetimeBalance(),
						UnsettledBalance: expectedChannel2.UnsettledBalance(),
					},
					PerHermes: map[common.Address]event.Earnings{
						hermesID: {
							LifetimeBalance:  expectedChannel2.LifetimeBalance(),
							UnsettledBalance: expectedChannel2.UnsettledBalance(),
						},
					},
				},
			},
			lastEvent,
		)
		return true
	}, 2*time.Second, 10*time.Millisecond)
}

func TestHermesChannelRepository_DataRace(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")
	channelID := common.HexToAddress("0x00000000000000000000000000000000000000003")
	beneficiary := common.HexToAddress("0x144")
	chainID_ := int64(1)

	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	publisher := mocks.NewEventBus()
	mockBeneficiaryProvider := &mockBeneficiaryProvider{
		b: beneficiary,
	}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, publisher, mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})

	var wg sync.WaitGroup

	channels := repo.List(chainID_)
	active := new(atomic.Bool)
	active.Store(true)

	promise := HermesPromise{ChannelID: channelID.Hex(), Identity: id, HermesID: hermesID}
	err := repo.updateChannelWithLatestPromise(chainID_, channelID.Hex(), id, hermesID, promise, true)
	assert.NoError(t, err)

	channels = repo.List(chainID_)

	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			promise := HermesPromise{ChannelID: channelID.Hex(), Identity: id, HermesID: hermesID}
			err := repo.updateChannelWithLatestPromise(chainID_, channelID.Hex(), id, hermesID, promise, true)
			assert.NoError(t, err)
		}
		active.Store(false)
	}()
	go func() {
		defer wg.Done()

		for active.Load() == true {
			// race between Get() and updateChannel()
			repo.Get(chainID_, id, hermesID)
		}
	}()
	go func() {
		defer wg.Done()

		var state []HermesChannel
		for active.Load() == true {
			if err := copier.CopyWithOption(&state, channels, copier.Option{DeepCopy: true}); err != nil {
				panic(err)
			}
		}
	}()
	wg.Wait()
}

func TestHermesChannelRepository_BeneficiaryReset(t *testing.T) {
	// given
	id := identity.FromAddress("0x0000000000000000000000000000000000000001")
	hermesID := common.HexToAddress("0x00000000000000000000000000000000000000002")
	channelID := common.HexToAddress("0x00000000000000000000000000000000000000003")
	beneficiary := common.HexToAddress("0x144")

	promiseProvider := &mockHermesPromiseStorage{}
	channelStatusProvider := &mockProviderChannelStatusProvider{
		channelToReturn: mockProviderChannel,
	}
	publisher := mocks.NewEventBus()
	mockBeneficiaryProvider := &mockBeneficiaryProvider{
		b: beneficiary,
	}
	mockHermesCaller := &mockHermesCaller{}
	addrProv := &mockAddressProvider{}
	repo := NewHermesChannelRepository(promiseProvider, channelStatusProvider, publisher, mockBeneficiaryProvider, mockHermesCaller, addrProv, signerFactory, &mockEncryptor{})

	// when
	promise := HermesPromise{ChannelID: channelID.Hex(), Identity: id, HermesID: hermesID}
	err := repo.updateChannelWithLatestPromise(1, channelID.Hex(), id, hermesID, promise, false)
	assert.NoError(t, err)
	hermesChannel, exists := repo.Get(1, id, hermesID)

	// then
	assert.True(t, exists)
	assert.Equal(t, beneficiary, hermesChannel.Beneficiary)

	// when
	err = repo.updateChannelWithLatestPromise(1, channelID.Hex(), id, hermesID, promise, false)
	assert.NoError(t, err)
	hermesChannel, exists = repo.Get(1, id, hermesID)

	// then
	assert.True(t, exists)
	assert.Equal(t, beneficiary, hermesChannel.Beneficiary)
}

type mockBeneficiaryProvider struct {
	b common.Address
}

func (ms *mockBeneficiaryProvider) GetBeneficiary(identity common.Address) (common.Address, error) {
	return ms.b, nil
}

var signerFactory = func(id identity.Identity) identity.Signer {
	return &mockSignerFactory{}
}

type mockSignerFactory struct {
}

func (s *mockSignerFactory) Sign(message []byte) (identity.Signature, error) {
	return identity.Signature{}, nil
}
