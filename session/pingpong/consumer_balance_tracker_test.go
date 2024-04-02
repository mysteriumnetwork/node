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
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/stretchr/testify/assert"
)

var mockMystSCaddress = common.HexToAddress("0x0")

var initialBalance = big.NewInt(100000000)

var defaultWaitTime = 3 * time.Second
var defaultWaitInterval = 20 * time.Millisecond

var defaultCfg = ConsumerBalanceTrackerConfig{
	FastSync: PollConfig{
		Interval: time.Second * 1,
		Timeout:  time.Second * 6,
	},
	LongSync: PollConfig{
		Interval: time.Minute,
	},
}

func TestConsumerBalanceTracker_Fresh_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	id2 := identity.FromAddress("0x000000002")
	assert.NotEqual(t, id1.Address, id2.Address)

	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		bus: bus,
		res: big.NewInt(0),
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
		mystBalanceToReturn: big.NewInt(0),
	}
	calc := mockAddressProvider{}

	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{}, &calc, &mockBlockchainProvider, defaultCfg)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:      id1,
		Status:  registry.Registered,
		ChainID: 1,
	})
	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:      id2,
		Status:  registry.RegistrationError,
		ChainID: 1,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id2).Uint64() == 0
	}, defaultWaitTime, defaultWaitInterval)

	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id2,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id2).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var promised = big.NewInt(100)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    1,
		ConsumerID: id1,
		Current:    promised,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(new(big.Int).Sub(initialBalance, promised)) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Fast_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	t.Run("Takes balance from hermes response", func(t *testing.T) {
		bus := eventbus.New()
		mcts := mockConsumerTotalsStorage{
			bus: bus,
		}
		bc := mockConsumerBalanceChecker{
			channelToReturn: client.ConsumerChannel{
				Balance: initialBalance,
				Settled: big.NewInt(0),
			},
		}
		calc := mockAddressProvider{}
		mockBlockchainProvider := mockBlockchainInfoProvider{}

		var ba = big.NewInt(10000000)
		cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: ba,
				ChainID:      1,
			},
		}, &mockRegistrationStatusProvider{}, &calc, &mockBlockchainProvider, defaultCfg)

		err := cbt.Subscribe(bus)
		assert.NoError(t, err)

		bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
			ID:      id1,
			Status:  registry.InProgress,
			ChainID: 1,
		})

		assert.Eventually(t, func() bool {
			return cbt.GetBalance(1, id1).Cmp(ba) == 0
		}, defaultWaitTime, defaultWaitInterval)
	})
	t.Run("Falls back to blockchain balance if no bounty is specified on transactor", func(t *testing.T) {
		t.Skip()
		bus := eventbus.New()
		mcts := mockConsumerTotalsStorage{
			res: big.NewInt(0),
			bus: bus,
		}
		var ba = big.NewInt(10000000)
		bc := mockConsumerBalanceChecker{
			channelToReturn: client.ConsumerChannel{
				Balance: initialBalance,
				Settled: big.NewInt(0),
			},
			mystBalanceToReturn: ba,
		}
		calc := mockAddressProvider{}
		mockBlockchainProvider := mockBlockchainInfoProvider{}

		cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: big.NewInt(0),
				ChainID:      1,
			},
		}, &mockRegistrationStatusProvider{}, &calc, &mockBlockchainProvider, defaultCfg)

		err := cbt.Subscribe(bus)
		assert.NoError(t, err)

		bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
			ID:      id1,
			Status:  registry.InProgress,
			ChainID: 1,
		})

		assert.Eventually(t, func() bool {
			return cbt.GetBalance(1, id1).Cmp(ba) == 0
		}, defaultWaitTime, defaultWaitInterval)
	})
}

func TestConsumerBalanceTracker_Handles_GrandTotalChanges(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised = big.NewInt(100)
	bus := eventbus.New()

	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
	}
	calc := mockAddressProvider{}
	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{grandTotalPromised, new(big.Int)}, &mockTransactor{}, &mockRegistrationStatusProvider{}, &calc, &mockBlockchainProvider, defaultCfg)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id1,
	})
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(new(big.Int).Sub(initialBalance, grandTotalPromised)) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var diff = big.NewInt(10)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    1,
		ConsumerID: id1,
		Current:    new(big.Int).Add(grandTotalPromised, diff),
	})

	assert.Eventually(t, func() bool {
		div := new(big.Int).Sub(initialBalance, grandTotalPromised)
		currentBalance := new(big.Int).Sub(div, diff)
		return cbt.GetBalance(1, id1).Cmp(currentBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var diff2 = big.NewInt(20)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ChainID:    1,
		ConsumerID: id1,
		Current:    new(big.Int).Add(grandTotalPromised, diff2),
	})

	assert.Eventually(t, func() bool {
		div := new(big.Int).Sub(initialBalance, grandTotalPromised)
		currentBalance := new(big.Int).Sub(div, diff2)
		return cbt.GetBalance(1, id1).Cmp(currentBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_FallsBackToTransactorIfInProgress(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		errToReturn:         errors.New("no contract deployed"),
		mystBalanceToReturn: initialBalance,
	}
	cfg := defaultCfg
	cfg.LongSync.Interval = time.Millisecond * 300
	calc := mockAddressProvider{}
	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{grandTotalPromised, new(big.Int)}, &mockTransactor{
		statusToReturn: registry.TransactorStatusResponse{
			Status:       registry.TransactorRegistrationEntryStatusCreated,
			ChainID:      1,
			BountyAmount: big.NewInt(100),
		},
	}, &mockRegistrationStatusProvider{
		map[string]mockRegistrationStatus{
			fmt.Sprintf("%d%s", 1, id1.Address): {
				status: registry.InProgress,
			},
		},
	}, &calc, &mockBlockchainProvider, cfg)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id1,
	})
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Uint64() == 100
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_UnregisteredBalanceReturned(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		mystBalanceToReturn: initialBalance,
		errToReturn:         errors.New("boom"),
	}
	calc := mockAddressProvider{}
	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{grandTotalPromised, new(big.Int)}, &mockTransactor{}, &mockRegistrationStatusProvider{
		map[string]mockRegistrationStatus{
			fmt.Sprintf("%d%s", 1, id1.Address): {
				status: registry.Unregistered,
			},
		},
	}, &calc, &mockBlockchainProvider, defaultCfg)

	b := cbt.ForceBalanceUpdate(1, id1)
	assert.Equal(t, initialBalance, b)
}

func TestConsumerBalanceTracker_InprogressUnregisteredBalanceReturnedWhenNoBounty(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		errToReturn:         errors.New("no contract deployed"),
		mystBalanceToReturn: initialBalance,
	}
	cfg := defaultCfg
	cfg.LongSync.Interval = time.Millisecond * 300
	calc := mockAddressProvider{}
	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{grandTotalPromised, new(big.Int)}, &mockTransactor{
		statusToReturn: registry.TransactorStatusResponse{
			Status:       registry.TransactorRegistrationEntryStatusCreated,
			ChainID:      1,
			BountyAmount: big.NewInt(0),
		},
	}, &mockRegistrationStatusProvider{
		map[string]mockRegistrationStatus{
			fmt.Sprintf("%d%s", 1, id1.Address): {
				status: registry.InProgress,
			},
		},
	}, &calc, &mockBlockchainProvider, cfg)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id1,
	})
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_RecoverGrandTotalPromisedSettledIsBiggerThanPromissedNotOffChain(t *testing.T) {
	// make data race more likely to happen
	for i := 0; i < 10; i++ {
		id1 := identity.FromAddress("0x000000001")
		grandTotalPromised := big.NewInt(10)
		settledAmount := big.NewInt(11)
		bus := eventbus.New()
		mcts := NewConsumerTotalsStorage(bus)
		bc := mockConsumerBalanceChecker{}
		cfg := defaultCfg
		cfg.LongSync.Interval = time.Millisecond * 300
		calc := newMockAddressProvider()
		calc.addrToReturn = id1.ToCommonAddress()
		mockBlockchainProvider := &mockBlockchainInfoProvider{}

		mockBlockchainProvider.AddConsumerChannelsHermes(1, id1.ToCommonAddress(), client.ConsumersHermes{
			Settled: big.NewInt(6),
		})

		cbt := NewConsumerBalanceTracker(bus, &bc, mcts, &mockconsumerInfoGetter{grandTotalPromised, settledAmount}, &mockTransactor{}, &mockRegistrationStatusProvider{}, calc, mockBlockchainProvider, defaultCfg)

		err := cbt.Subscribe(bus)
		assert.NoError(t, err)
		bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
			ChainID: 1,
			ID:      id1,
		})
		assert.Eventually(t, func() bool {
			savedBalance, _ := mcts.Get(1, id1, common.BigToAddress(big.NewInt(0)))
			return savedBalance.Cmp(big.NewInt(6)) == 0
		}, defaultWaitTime, defaultWaitInterval)
	}
}

type mockConsumerBalanceChecker struct {
	channelToReturn client.ConsumerChannel
	errToReturn     error
	errLock         sync.Mutex

	mystBalanceToReturn *big.Int
	mystBalanceError    error
}

func (mcbc *mockConsumerBalanceChecker) getError() error {
	mcbc.errLock.Lock()
	defer mcbc.errLock.Unlock()
	return mcbc.errToReturn
}

func (mcbc *mockConsumerBalanceChecker) setError(err error) {
	mcbc.errLock.Lock()
	defer mcbc.errLock.Unlock()
	mcbc.errToReturn = err
}

func (mcbc *mockConsumerBalanceChecker) GetConsumerChannel(chainID int64, addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error) {
	return mcbc.channelToReturn, mcbc.getError()
}

func (mcbc *mockConsumerBalanceChecker) GetMystBalance(chainID int64, mystAddress, identity common.Address) (*big.Int, error) {
	return mcbc.mystBalanceToReturn, mcbc.mystBalanceError
}

type mockconsumerInfoGetter struct {
	amount  *big.Int
	settled *big.Int
}

func (mcig *mockconsumerInfoGetter) GetConsumerData(_ int64, _ string, _ time.Duration) (HermesUserInfo, error) {
	return HermesUserInfo{
		Settled: mcig.settled,
		LatestPromise: LatestPromise{
			Amount: mcig.amount,
		},
	}, nil
}

func TestConsumerBalanceTracker_DoesNotBlockedOnEmptyBalancesList(t *testing.T) {
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{bus: bus, res: big.NewInt(0)}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
	}
	calc := mockAddressProvider{}
	mockBlockchainProvider := mockBlockchainInfoProvider{}

	cbt := NewConsumerBalanceTracker(bus, &bc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{}, &calc, &mockBlockchainProvider, defaultCfg)

	// Make sure we are not dead locked here. https://github.com/mysteriumnetwork/node/issues/2181
	cbt.updateGrandTotal(1, identity.FromAddress("0x0000"), big.NewInt(1))
}

func TestConsumerBalance_GetBalance(t *testing.T) {
	type fields struct {
		BCBalance          *big.Int
		BCSettled          *big.Int
		GrandTotalPromised *big.Int
	}
	tests := []struct {
		name   string
		fields fields
		want   *big.Int
	}{
		{
			name: "handles bc balance underflow",
			fields: fields{
				BCBalance:          big.NewInt(0),
				BCSettled:          big.NewInt(0),
				GrandTotalPromised: big.NewInt(1),
			},
			want: big.NewInt(0),
		},
		{
			name: "handles grand total underflow",
			fields: fields{
				BCBalance:          big.NewInt(0),
				BCSettled:          big.NewInt(1),
				GrandTotalPromised: big.NewInt(0),
			},
			want: big.NewInt(0),
		},
		{
			name: "calculates balance correctly",
			fields: fields{
				BCBalance:          big.NewInt(3),
				BCSettled:          big.NewInt(1),
				GrandTotalPromised: big.NewInt(2),
			},
			want: big.NewInt(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := ConsumerBalance{
				BCBalance:          tt.fields.BCBalance,
				BCSettled:          tt.fields.BCSettled,
				GrandTotalPromised: tt.fields.GrandTotalPromised,
			}
			if got := cb.GetBalance(); got.Cmp(tt.want) != 0 {
				t.Errorf("ConsumerBalance.GetBalance() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockAddressProvider struct {
	transactor   common.Address
	addrToReturn common.Address
	channels     map[string]common.Address
}

func newMockAddressProvider() *mockAddressProvider {
	return &mockAddressProvider{channels: make(map[string]common.Address)}
}

func (ma *mockAddressProvider) GetActiveChannelAddress(chainID int64, id common.Address) (common.Address, error) {
	return ma.addrToReturn, nil
}
func (ma *mockAddressProvider) GetActiveChannelImplementation(chainID int64) (common.Address, error) {
	return common.Address{}, nil
}
func (ma *mockAddressProvider) GetMystAddress(chainID int64) (common.Address, error) {
	return common.Address{}, nil
}
func (ma *mockAddressProvider) GetActiveHermes(chainID int64) (common.Address, error) {
	return common.Address{}, nil
}
func (ma *mockAddressProvider) GetRegistryAddress(chainID int64) (common.Address, error) {
	return common.Address{}, nil
}
func (ma *mockAddressProvider) GetArbitraryChannelAddress(hermes, registry, channel common.Address, id common.Address) (common.Address, error) {
	return ma.addrToReturn, nil
}
func (ma *mockAddressProvider) GetChannelImplementationForHermes(chainID int64, hermes common.Address) (common.Address, error) {
	return common.Address{}, nil
}
func (ma *mockAddressProvider) GetKnownHermeses(chainID int64) ([]common.Address, error) {
	return []common.Address{ma.addrToReturn}, nil
}
func (ma *mockAddressProvider) GetHermesChannelAddress(chainID int64, id, hermesAddr common.Address) (common.Address, error) {
	channelAddr, _ := ma.channels[fmt.Sprintf("%d-%s", chainID, id)]
	return channelAddr, nil
}

func (ma *mockAddressProvider) setChannelAddress(chainID int64, id, channelAddr common.Address) {
	ma.channels[fmt.Sprintf("%d-%s", chainID, id)] = channelAddr
}

var _ blockchainInfoProvider = (*mockBlockchainInfoProvider)(nil)

type mockBlockchainInfoProvider struct {
	consumerChannelsHermesMap map[string]client.ConsumersHermes
}

func (p *mockBlockchainInfoProvider) GetConsumerChannelsHermes(chainID int64, channelAddress common.Address) (client.ConsumersHermes, error) {
	result, ok := p.consumerChannelsHermesMap[p.mapKey(chainID, channelAddress)]
	if !ok {
		return client.ConsumersHermes{}, fmt.Errorf("mock consumer channels hermes not found")
	}

	return result, nil
}

func (p *mockBlockchainInfoProvider) AddConsumerChannelsHermes(chainID int64, channelAddress common.Address, consumerHermes client.ConsumersHermes) {
	if p.consumerChannelsHermesMap == nil {
		p.consumerChannelsHermesMap = map[string]client.ConsumersHermes{}
	}

	p.consumerChannelsHermesMap[p.mapKey(chainID, channelAddress)] = consumerHermes
}

func (p *mockBlockchainInfoProvider) mapKey(chainID int64, channelAddress common.Address) string {
	return fmt.Sprintf("%d_%s", chainID, channelAddress.String())
}
