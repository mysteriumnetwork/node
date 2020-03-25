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
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/stretchr/testify/assert"
)

var mockMystSCaddress = common.HexToAddress("0x0")

const initialBalance = 100000000

var defaultWaitTime = time.Millisecond * 50
var defaultWaitInterval = time.Millisecond

func TestConsumerBalanceTracker_Fresh_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	id2 := identity.FromAddress("0x000000002")
	assert.NotEqual(t, id1.Address, id2.Address)

	mcts := mockConsumerTotalsStorage{}
	bus := eventbus.New()
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, mockMystSCaddress, &bc, &calc, &mcts)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:     id1,
		Status: registry.RegisteredProvider,
	})
	bus.Publish(registry.AppTopicIdentityRegistration, registry.AppEventIdentityRegistration{
		ID:     id2,
		Status: registry.RegistrationError,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance
	}, defaultWaitTime, defaultWaitInterval)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id2) == 0
	}, defaultWaitTime, defaultWaitInterval)

	bus.Publish(identity.AppTopicIdentityUnlock, id2.Address)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id2) == initialBalance
	}, defaultWaitTime, defaultWaitInterval)

	var promised uint64 = 100
	bus.Publish(AppTopicExchangeMessage, AppEventExchangeMessage{
		Identity:       id1,
		AmountPromised: promised,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-promised
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Handles_RRecovery(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	mcts := mockConsumerTotalsStorage{}
	bus := eventbus.New()
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, mockMystSCaddress, &bc, &calc, &mcts)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance
	}, defaultWaitTime, defaultWaitInterval)

	var changedPromised uint64 = 232323
	mcts.setResult(changedPromised)

	bus.Publish(AppTopicGrandTotalRecovered, GrandTotalRecovered{
		Identity: id1,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-changedPromised
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Handles_Promises(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised uint64 = 100
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
	}
	bus := eventbus.New()
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, mockMystSCaddress, &bc, &calc, &mcts)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised
	}, defaultWaitTime, defaultWaitInterval)

	var diff uint64 = 1
	bus.Publish(AppTopicExchangeMessage, AppEventExchangeMessage{
		Identity:       id1,
		AmountPromised: diff,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised-diff
	}, defaultWaitTime, defaultWaitInterval)

	var diff2 uint64 = 4
	bus.Publish(AppTopicExchangeMessage, AppEventExchangeMessage{
		Identity:       id1,
		AmountPromised: diff2,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised-diff2-diff
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Handles_TopUp(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	var grandTotalPromised uint64 = 100
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
	}
	bus := eventbus.New()
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
		ch: make(chan *bindings.MystTokenTransfer),
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, mockMystSCaddress, &bc, &calc, &mcts)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised
	}, defaultWaitTime, defaultWaitInterval)

	var diff uint64 = 1
	bus.Publish(AppTopicExchangeMessage, AppEventExchangeMessage{
		Identity:       id1,
		AmountPromised: diff,
	})

	bus.Publish(registry.AppTopicTransactorTopUp, id1.Address)
	var topUpAmount uint64 = 123
	bc.ch <- &bindings.MystTokenTransfer{
		Value: big.NewInt(0).SetUint64(topUpAmount),
	}

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised-diff+topUpAmount
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_increaseBalance(t *testing.T) {
	type fields struct {
		balances map[identity.Identity]uint64
	}
	type args struct {
		id identity.Identity
		b  uint64
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		expectedBalance uint64
	}{
		{
			name: "defaults to new balance",
			fields: fields{
				balances: make(map[identity.Identity]uint64),
			},
			args: args{
				id: mockID,
				b:  100000,
			},
			expectedBalance: 100000,
		},
		{
			name: "adds to existing balance",
			fields: fields{
				balances: map[identity.Identity]uint64{mockID: 100000},
			},
			args: args{
				id: mockID,
				b:  100000,
			},
			expectedBalance: 200000,
		},
		{
			name: "returns 0 on overflow",
			fields: fields{
				balances: map[identity.Identity]uint64{mockID: math.MaxUint64},
			},
			args: args{
				id: mockID,
				b:  1,
			},
			expectedBalance: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbt := &ConsumerBalanceTracker{
				balances:  tt.fields.balances,
				publisher: eventbus.New(),
			}
			cbt.increaseBalance(tt.args.id, tt.args.b)
			res := cbt.GetBalance(tt.args.id)
			assert.Equal(t, tt.expectedBalance, res)
		})
	}
}

func TestConsumerBalanceTracker_decreaseBalance(t *testing.T) {
	type fields struct {
		balances map[identity.Identity]uint64
	}
	type args struct {
		id identity.Identity
		b  uint64
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		expectedBalance uint64
	}{
		{
			name: "returns 0 on underflow",
			fields: fields{
				balances: map[identity.Identity]uint64{identityOne: 1},
			},
			args: args{
				id: identityOne,
				b:  2,
			},
			expectedBalance: 0,
		},
		{
			name: "handles non existing identity",
			fields: fields{
				balances: make(map[identity.Identity]uint64),
			},
			args: args{
				id: identityOne,
				b:  2,
			},
			expectedBalance: 0,
		},
		{
			name: "subtracts correctly",
			fields: fields{
				balances: map[identity.Identity]uint64{identityOne: 100},
			},
			args: args{
				id: identityOne,
				b:  2,
			},
			expectedBalance: 98,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbt := &ConsumerBalanceTracker{
				balances:  tt.fields.balances,
				publisher: eventbus.New(),
			}
			cbt.decreaseBalance(tt.args.id, tt.args.b)
			assert.Equal(t, tt.expectedBalance, cbt.balances[tt.args.id])
		})
	}
}

type mockAccountantBalanceFetcher struct {
	consumerData ConsumerData
	err          error
}

func (mabf *mockAccountantBalanceFetcher) GetConsumerData(channel string) (ConsumerData, error) {
	return mabf.consumerData, mabf.err
}

type mockConsumerBalanceChecker struct {
	channelToReturn client.ConsumerChannel
	errToReturn     error
	ch              chan *bindings.MystTokenTransfer
}

func (mcbc *mockConsumerBalanceChecker) GetConsumerChannel(addr common.Address, mystSCAddress common.Address) (client.ConsumerChannel, error) {
	return mcbc.channelToReturn, mcbc.errToReturn
}

func (mcbc *mockConsumerBalanceChecker) SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error) {
	return mcbc.ch, func() {}, nil
}

type mockChannelAddressCalculator struct {
	addrToReturn common.Address
	errToReturn  error
}

func (mcac *mockChannelAddressCalculator) GetChannelAddress(id identity.Identity) (common.Address, error) {
	return mcac.addrToReturn, mcac.errToReturn
}
