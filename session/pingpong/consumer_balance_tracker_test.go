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
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/stretchr/testify/assert"
)

var mockMystSCaddress = common.HexToAddress("0x0")

var initialBalance = big.NewInt(100000000)

var defaultWaitTime = 2 * time.Second
var defaultWaitInterval = 10 * time.Millisecond

func TestConsumerBalanceTracker_Fresh_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	id2 := identity.FromAddress("0x000000002")
	hermesID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
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
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{})

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
	hermesID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
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
		calc := mockChannelAddressCalculator{}

		var ba = big.NewInt(10000000)
		cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: ba,
				ChainID:      1,
			},
		}, &mockRegistrationStatusProvider{})

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
		calc := mockChannelAddressCalculator{}

		cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: big.NewInt(0),
				ChainID:      1,
			},
		}, &mockRegistrationStatusProvider{})

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

func TestConsumerBalanceTracker_Handles_FreeRegistration(t *testing.T) {
	fees := big.NewInt(10000000)
	bounty := big.NewInt(100000000)

	id1 := identity.FromAddress("0x000000001")
	hermesID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
	var grandTotalPromised = big.NewInt(100)
	bus := eventbus.New()

	updates := make(chan *bindings.MystTokenTransfer, 0)
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
	}
	bc := mockConsumerBalanceChecker{
		ch: updates,
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(0),
			Settled: big.NewInt(0),
		},
	}

	tr := &mockTransactor{
		feesToReturn: registry.FeesResponse{Fee: fees},
		statusToReturn: registry.TransactorStatusResponse{
			Status:       registry.TransactorRegistrationEntryStatusCreated,
			BountyAmount: bounty,
			ChainID:      1,
		},
	}

	rp := &mockRegistrationStatusProvider{
		identities: map[string]mockRegistrationStatus{
			"1" + id1.Address: {status: registry.InProgress},
		},
	}

	channel := common.HexToAddress("0x000000002")
	calc := mockChannelAddressCalculator{
		addrToReturn: channel,
	}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, tr, rp)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id1,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(bounty) == 0
	}, defaultWaitTime, defaultWaitInterval, "balance should be equal to the bounty amount")

	updates <- &bindings.MystTokenTransfer{
		From:  transactorID,
		To:    channel,
		Value: new(big.Int).Add(bounty, fees),
	}

	assert.Never(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(new(big.Int).Add(bounty, fees)) == 0
	}, defaultWaitTime, defaultWaitInterval, "transactor free registration transaction should affect increase balance")

	updates <- &bindings.MystTokenTransfer{
		From:  transactorID,
		To:    channel,
		Value: new(big.Int).Set(fees),
	}

	bounty = new(big.Int).Add(bounty, fees)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(bounty) == 0
	}, defaultWaitTime, defaultWaitInterval, "transactor fee decrease should affect balance")

	updates <- &bindings.MystTokenTransfer{
		From:  common.HexToAddress("0x000000002"),
		To:    channel,
		Value: new(big.Int).Set(fees),
	}

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(new(big.Int).Add(bounty, fees)) == 0
	}, defaultWaitTime, defaultWaitInterval, "if transaction was received not from transactor, it should affect balance")
}

func TestConsumerBalanceTracker_Handles_GrandTotalChanges(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	hermesID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
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
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{}, &mockRegistrationStatusProvider{})

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
	accountantID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
		ch: make(chan *bindings.MystTokenTransfer),
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{
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
	})

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

func TestConsumerBalanceTracker_ForceUpdatesOnSuccessfulSubscription(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	accountantID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		mystBalanceToReturn: new(big.Int),
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
		errToReturn: errors.New("boom"),
		ch:          make(chan *bindings.MystTokenTransfer),
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{}, &mockRegistrationStatusProvider{
		map[string]mockRegistrationStatus{
			fmt.Sprintf("%d%s", 1, id1.Address): {
				status: registry.Unregistered,
			},
		},
	})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{
		ChainID: 1,
		ID:      id1,
	})

	time.Sleep(time.Millisecond * 20)
	bc.setError(nil)

	bc.ch <- &bindings.MystTokenTransfer{
		Value: initialBalance,
	}

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(1, id1).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_UnregisteredBalanceReturned(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	accountantID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")
	var grandTotalPromised = new(big.Int)
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		mystBalanceToReturn: initialBalance,
		errToReturn:         errors.New("boom"),
		ch:                  make(chan *bindings.MystTokenTransfer),
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{}, &mockRegistrationStatusProvider{
		map[string]mockRegistrationStatus{
			fmt.Sprintf("%d%s", 1, id1.Address): {
				status: registry.Unregistered,
			},
		},
	})

	b := cbt.ForceBalanceUpdate(1, id1)
	assert.Equal(t, initialBalance, b)
}

type mockConsumerBalanceChecker struct {
	channelToReturn client.ConsumerChannel
	errToReturn     error
	errLock         sync.Mutex
	ch              chan *bindings.MystTokenTransfer

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

func (mcbc *mockConsumerBalanceChecker) SubscribeToConsumerBalanceEvent(chainID int64, channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error) {
	return mcbc.ch, func() {}, nil
}

func (mcbc *mockConsumerBalanceChecker) GetMystBalance(chainID int64, mystAddress, identity common.Address) (*big.Int, error) {
	return mcbc.mystBalanceToReturn, mcbc.mystBalanceError
}

type mockChannelAddressCalculator struct {
	addrToReturn common.Address
	errToReturn  error
}

func (mcac *mockChannelAddressCalculator) GetChannelAddress(id identity.Identity) (common.Address, error) {
	return mcac.addrToReturn, mcac.errToReturn
}

type mockconsumerInfoGetter struct {
	amount *big.Int
}

func (mcig *mockconsumerInfoGetter) GetConsumerData(_ int64, _ string) (ConsumerData, error) {
	return ConsumerData{
		LatestPromise: LatestPromise{
			Amount: mcig.amount,
		},
	}, nil
}

func TestConsumerBalanceTracker_DoesNotBlockedOnEmptyBalancesList(t *testing.T) {
	hermesID := common.HexToAddress("0x000000acc")
	transactorID := common.HexToAddress("0x000003")

	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{bus: bus, res: big.NewInt(0)}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, transactorID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{})

	// Make sure we are not dead locked here. https://github.com/mysteriumnetwork/node/issues/2181
	cbt.increaseBCBalance(1, identity.FromAddress("0x0000"), big.NewInt(1))
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
