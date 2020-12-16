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
	"testing"

	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHermesChannel_balance(t *testing.T) {
	channel := HermesChannel{
		Channel: client.ProviderChannel{
			Stake:   big.NewInt(100),
			Settled: big.NewInt(10),
		},
		lastPromise: HermesPromise{
			Promise: crypto.Promise{Amount: big.NewInt(15)},
		},
	}
	assert.Equal(t, big.NewInt(110), channel.availableBalance())
	assert.Equal(t, big.NewInt(95), channel.balance())
	assert.Equal(t, big.NewInt(5), channel.UnsettledBalance())

	channel = HermesChannel{
		Channel: client.ProviderChannel{
			Stake:   big.NewInt(100),
			Settled: big.NewInt(10),
		},
		lastPromise: HermesPromise{
			Promise: crypto.Promise{Amount: big.NewInt(16)},
		},
	}
	assert.Equal(t, big.NewInt(110), channel.availableBalance())
	assert.Equal(t, big.NewInt(94), channel.balance())
	assert.Equal(t, big.NewInt(6), channel.UnsettledBalance())
}
