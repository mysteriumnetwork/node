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
	cbt := NewConsumerBalanceTracker(bus, mockMystSCaddress, &bc, &calc)

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

type mockConsumerBalanceChecker struct {
	amountToReturn *big.Int
	errToReturn    error
}

func (mcbc *mockConsumerBalanceChecker) GetConsumerBalance(channel, mystSCAddress common.Address) (*big.Int, error) {
	return mcbc.amountToReturn, mcbc.errToReturn
}

type mockChannelAddressCalculator struct {
	addrToReturn common.Address
	errToReturn  error
}

func (mcac *mockChannelAddressCalculator) GetChannelAddress(id identity.Identity) (common.Address, error) {
	return mcac.addrToReturn, mcac.errToReturn
}
