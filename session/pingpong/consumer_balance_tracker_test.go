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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/stretchr/testify/assert"
)

var mockMystSCaddress = common.HexToAddress("0x0")

const initialBalance = 100000000

func TestConsumerBalanceTracker(t *testing.T) {
	id1 := identity.FromAddress("0x000000001")
	id2 := identity.FromAddress("0x000000002")
	assert.NotEqual(t, id1.Address, id2.Address)

	bus := eventbus.New()
	bc := mockConsumerBalanceChecker{
		amountToReturn: big.NewInt(initialBalance),
	}
	calc := mockChannelAddressCalculator{}

	balanceFetcher := &mockAccountantBalanceFetcher{consumerData: ConsumerData{
		Balance:  initialBalance,
		Promised: 0,
	}}

	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, &bc, &calc, balanceFetcher.GetConsumerData)

	err := cbt.Subscribe(bus)
	assert.NoError(t, err)

	bus.Publish(registry.RegistrationEventTopic, registry.RegistrationEventPayload{
		ID:     id1,
		Status: registry.RegisteredProvider,
	})
	bus.Publish(registry.RegistrationEventTopic, registry.RegistrationEventPayload{
		ID:     id2,
		Status: registry.RegistrationError,
	})

	err = waitForBalance(cbt, id1, initialBalance)
	assert.Nil(t, err)

	err = waitForBalance(cbt, id2, 0)
	assert.Nil(t, err)

	bus.Publish(identity.IdentityUnlockTopic, id2.Address)

	err = waitForBalance(cbt, id2, initialBalance)
	assert.Nil(t, err)

	var promised uint64 = 100
	bus.Publish(ExchangeMessageTopic, ExchangeMessageEventPayload{
		Identity:       id1,
		AmountPromised: promised,
	})

	err = waitForBalance(cbt, id1, initialBalance-promised)
	assert.Nil(t, err)
}

func waitForBalance(balanceTracker *ConsumerBalanceTracker, id identity.Identity, balance uint64) error {
	timer := time.NewTimer(time.Millisecond)
	for i := 0; i < 20; i++ {
		select {
		case <-timer.C:
			b := balanceTracker.GetBalance(id)
			if b == balance {
				return nil
			}
			timer.Reset(time.Millisecond)
		}
	}
	return errors.New("did not get balance in time")
}

func TestConsumerBalanceTracker_UpdateAccountantBalance(t *testing.T) {
	type fields struct {
		balances                 map[identity.Identity]Balance
		accountantBalanceFetcher func(id string) (ConsumerData, error)
	}
	type args struct {
		id identity.Identity
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		expectedBalance  uint64
		expectedEstimate uint64
	}{
		{
			name: "set balance to an unknown identity",
			fields: fields{
				balances: make(map[identity.Identity]Balance),
				accountantBalanceFetcher: func(id string) (ConsumerData, error) {
					return ConsumerData{
						Balance: 100,
					}, nil
				},
			},
			args: args{
				id: mockID,
			},
			expectedBalance:  100,
			expectedEstimate: 100,
		},
		{
			name: "increases balance to an known identity",
			fields: fields{
				balances: map[identity.Identity]Balance{
					mockID: Balance{
						BCBalance:       1020,
						CurrentEstimate: 1010,
					},
				},
				accountantBalanceFetcher: func(id string) (ConsumerData, error) {
					return ConsumerData{
						Balance: 1100,
					}, nil
				},
			},
			args: args{
				id: mockID,
			},
			expectedBalance:  1100,
			expectedEstimate: 1090,
		},
		{
			name: "decreases balance to an known identity",
			fields: fields{
				balances: map[identity.Identity]Balance{
					mockID: Balance{
						BCBalance:       1020,
						CurrentEstimate: 1010,
					},
				},
				accountantBalanceFetcher: func(id string) (ConsumerData, error) {
					return ConsumerData{
						Balance: 1000,
					}, nil
				},
			},
			args: args{
				id: mockID,
			},
			expectedBalance:  1000,
			expectedEstimate: 990,
		},
		{
			name: "ignores errors, sets nothing",
			fields: fields{
				balances: make(map[identity.Identity]Balance),
				accountantBalanceFetcher: func(id string) (ConsumerData, error) {
					return ConsumerData{}, errors.New("explosions")
				},
			},
			args: args{
				id: mockID,
			},
			expectedBalance:  0,
			expectedEstimate: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbt := &ConsumerBalanceTracker{
				balances:                 tt.fields.balances,
				publisher:                eventbus.New(),
				accountantBalanceFetcher: tt.fields.accountantBalanceFetcher,
			}

			cbt.updateBalanceFromAccountant(tt.args.id)
			res := cbt.balances[tt.args.id]
			assert.Equal(t, tt.expectedBalance, res.BCBalance)
			assert.Equal(t, tt.expectedEstimate, res.CurrentEstimate)
		})
	}
}

func TestConsumerBalanceTracker_increaseBalance(t *testing.T) {
	type fields struct {
		balances map[identity.Identity]Balance
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
				balances: make(map[identity.Identity]Balance),
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
				balances: map[identity.Identity]Balance{mockID: Balance{
					BCBalance:       100000,
					CurrentEstimate: 100000,
				}},
			},
			args: args{
				id: mockID,
				b:  100000,
			},
			expectedBalance: 200000,
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

type mockAccountantBalanceFetcher struct {
	consumerData ConsumerData
	err          error
}

func (mabf *mockAccountantBalanceFetcher) GetConsumerData(channel string) (ConsumerData, error) {
	return mabf.consumerData, mabf.err
}

type mockConsumerBalanceChecker struct {
	amountToReturn *big.Int
	errToReturn    error
}

func (mcbc *mockConsumerBalanceChecker) GetConsumerBalance(channel, mystSCAddress common.Address) (*big.Int, error) {
	return mcbc.amountToReturn, mcbc.errToReturn
}

func (mcbc *mockConsumerBalanceChecker) SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address) (chan *bindings.MystTokenTransfer, func(), error) {
	return nil, nil, nil
}

type mockChannelAddressCalculator struct {
	addrToReturn common.Address
	errToReturn  error
}

func (mcac *mockChannelAddressCalculator) GetChannelAddress(id identity.Identity) (common.Address, error) {
	return mcac.addrToReturn, mcac.errToReturn
}
