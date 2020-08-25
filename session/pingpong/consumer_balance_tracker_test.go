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
	"math/big"
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

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:     id1,
		Status: registry.Registered,
	})
	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:     id2,
		Status: registry.RegistrationError,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id2).Uint64() == 0
	}, defaultWaitTime, defaultWaitInterval)

	bus.Publish(identity.AppTopicIdentityUnlock, id2.Address)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id2).Cmp(initialBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var promised = big.NewInt(100)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    promised,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1).Cmp(new(big.Int).Sub(initialBalance, promised)) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Fast_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	hermesID := common.HexToAddress("0x000000acc")
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
		cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: ba,
			},
		}, &mockRegistrationStatusProvider{})

		err := cbt.Subscribe(bus)
		assert.NoError(t, err)

		bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
			ID:     id1,
			Status: registry.InProgress,
		})

		assert.Eventually(t, func() bool {
			return cbt.GetBalance(id1).Cmp(ba) == 0
		}, defaultWaitTime, defaultWaitInterval)
	})
	t.Run("Falls back to blockchain balance if no bounty is specified on transactor", func(t *testing.T) {
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

		cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{
			statusToReturn: registry.TransactorStatusResponse{
				Status:       registry.TransactorRegistrationEntryStatusCreated,
				BountyAmount: big.NewInt(0),
			},
		}, &mockRegistrationStatusProvider{})

		err := cbt.Subscribe(bus)
		assert.NoError(t, err)

		bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
			ID:     id1,
			Status: registry.InProgress,
		})

		assert.Eventually(t, func() bool {
			return cbt.GetBalance(id1).Cmp(ba) == 0
		}, defaultWaitTime, defaultWaitInterval)
	})
}

func TestConsumerBalanceTracker_Handles_GrandTotalChanges(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	hermesID := common.HexToAddress("0x000000acc")
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
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{}, &mockRegistrationStatusProvider{})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1).Cmp(new(big.Int).Sub(initialBalance, grandTotalPromised)) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var diff = big.NewInt(10)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    new(big.Int).Add(grandTotalPromised, diff),
	})

	assert.Eventually(t, func() bool {
		div := new(big.Int).Sub(initialBalance, grandTotalPromised)
		currentBalance := new(big.Int).Sub(div, diff)
		return cbt.GetBalance(id1).Cmp(currentBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)

	var diff2 = big.NewInt(20)
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    new(big.Int).Add(grandTotalPromised, diff2),
	})

	assert.Eventually(t, func() bool {
		div := new(big.Int).Sub(initialBalance, grandTotalPromised)
		currentBalance := new(big.Int).Sub(div, diff2)
		return cbt.GetBalance(id1).Cmp(currentBalance) == 0
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_FallsBackToTransactorIfInProgress(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	accountantID := common.HexToAddress("0x000000acc")
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
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised}, &mockTransactor{
		statusToReturn: registry.TransactorStatusResponse{
			Status:       registry.TransactorRegistrationEntryStatusCreated,
			BountyAmount: big.NewInt(100),
		},
	}, &mockRegistrationStatusProvider{
		map[identity.Identity]mockRegistrationStatus{
			id1: {
				status: registry.InProgress,
			},
		},
	})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1).Uint64() == 100
	}, defaultWaitTime, defaultWaitInterval)
}

type mockConsumerBalanceChecker struct {
	channelToReturn client.ConsumerChannel
	errToReturn     error
	ch              chan *bindings.MystTokenTransfer

	mystBalanceToReturn *big.Int
	mystBalanceError    error
}

func (mcbc *mockConsumerBalanceChecker) GetConsumerChannel(addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error) {
	return mcbc.channelToReturn, mcbc.errToReturn
}

func (mcbc *mockConsumerBalanceChecker) SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error) {
	return mcbc.ch, func() {}, nil
}

func (mcbc *mockConsumerBalanceChecker) GetMystBalance(mystAddress, identity common.Address) (*big.Int, error) {
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

func (mcig *mockconsumerInfoGetter) GetConsumerData(_ string) (ConsumerData, error) {
	return ConsumerData{
		LatestPromise: LatestPromise{
			Amount: mcig.amount,
		},
	}, nil
}

func TestConsumerBalanceTracker_DoesNotBlockedOnEmptyBalancesList(t *testing.T) {
	hermesID := common.HexToAddress("0x000000acc")

	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{bus: bus, res: big.NewInt(0)}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: initialBalance,
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, hermesID, &bc, &calc, &mcts, &mockconsumerInfoGetter{}, &mockTransactor{}, &mockRegistrationStatusProvider{})

	// Make sure we are not dead locked here. https://github.com/mysteriumnetwork/node/issues/2181
	cbt.increaseBCBalance(identity.FromAddress("0x0000"), big.NewInt(1))
	cbt.updateGrandTotal(identity.FromAddress("0x0000"), big.NewInt(1))
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
