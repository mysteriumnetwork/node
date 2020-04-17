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

const initialBalance = 100000000

var defaultWaitTime = time.Millisecond * 50
var defaultWaitInterval = time.Millisecond

func TestConsumerBalanceTracker_Fresh_Registration(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	id2 := identity.FromAddress("0x000000002")
	accountantID := common.HexToAddress("0x000000acc")
	assert.NotEqual(t, id1.Address, id2.Address)

	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, &bc, &calc, &mcts, &mockconsumerInfoGetter{})

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
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    promised,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-promised
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Handles_GrandTotalChanges(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	accountantID := common.HexToAddress("0x000000acc")
	var grandTotalPromised uint64 = 100
	bus := eventbus.New()

	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised
	}, defaultWaitTime, defaultWaitInterval)

	var diff uint64 = 10
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    grandTotalPromised + diff,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised-diff
	}, defaultWaitTime, defaultWaitInterval)

	var diff2 uint64 = 20
	bus.Publish(event.AppTopicGrandTotalChanged, event.AppEventGrandTotalChanged{
		ConsumerID: id1,
		Current:    grandTotalPromised + diff2,
	})

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised-diff2
	}, defaultWaitTime, defaultWaitInterval)
}

func TestConsumerBalanceTracker_Handles_TopUp(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	accountantID := common.HexToAddress("0x000000acc")
	var grandTotalPromised uint64 = 100
	bus := eventbus.New()
	mcts := mockConsumerTotalsStorage{
		res: grandTotalPromised,
		bus: bus,
	}
	bc := mockConsumerBalanceChecker{
		channelToReturn: client.ConsumerChannel{
			Balance: big.NewInt(initialBalance),
			Settled: big.NewInt(0),
		},
		ch: make(chan *bindings.MystTokenTransfer),
	}
	calc := mockChannelAddressCalculator{}
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, accountantID, &bc, &calc, &mcts, &mockconsumerInfoGetter{grandTotalPromised})

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(identity.AppTopicIdentityUnlock, id1.Address)
	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised
	}, defaultWaitTime, defaultWaitInterval)

	bus.Publish(registry.AppTopicTransactorTopUp, id1.Address)
	var topUpAmount uint64 = 123
	bc.ch <- &bindings.MystTokenTransfer{
		Value: big.NewInt(0).SetUint64(topUpAmount),
	}

	assert.Eventually(t, func() bool {
		return cbt.GetBalance(id1) == initialBalance-grandTotalPromised+topUpAmount
	}, defaultWaitTime, defaultWaitInterval)
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

type mockconsumerInfoGetter struct {
	amount uint64
}

func (mcig *mockconsumerInfoGetter) GetConsumerData(_ string) (ConsumerData, error) {
	return ConsumerData{
		LatestPromise: LatestPromise{
			Amount: mcig.amount,
		},
	}, nil
}

func TestConsumerBalance_GetBalance(t *testing.T) {
	type fields struct {
		BCBalance          uint64
		BCSettled          uint64
		GrandTotalPromised uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   uint64
	}{
		{
			name: "handles bc balance underflow",
			fields: fields{
				BCBalance:          0,
				BCSettled:          0,
				GrandTotalPromised: 1,
			},
			want: 0,
		},
		{
			name: "handles grand total underflow",
			fields: fields{
				BCBalance:          0,
				BCSettled:          1,
				GrandTotalPromised: 0,
			},
			want: 0,
		},
		{
			name: "calculates balance correctly",
			fields: fields{
				BCBalance:          3,
				BCSettled:          1,
				GrandTotalPromised: 2,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := ConsumerBalance{
				BCBalance:          tt.fields.BCBalance,
				BCSettled:          tt.fields.BCSettled,
				GrandTotalPromised: tt.fields.GrandTotalPromised,
			}
			if got := cb.GetBalance(); got != tt.want {
				t.Errorf("ConsumerBalance.GetBalance() = %v, want %v", got, tt.want)
			}
		})
	}
}
