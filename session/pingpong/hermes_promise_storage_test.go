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
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHermesPromiseStorage(t *testing.T) {
	dir, err := os.MkdirTemp("", "hermesPromiseStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	hermesStorage := NewHermesPromiseStorage(bolt)

	id := identity.FromAddress("0x44440954558C5bFA0D4153B0002B1d1E3E3f5Ff5")
	firstHermes := common.HexToAddress("0x000000acc1")
	secondHermes := common.HexToAddress("0x000000acc2")

	firstPromise := HermesPromise{
		ChannelID:   "1",
		Identity:    id,
		HermesID:    firstHermes,
		Promise:     crypto.Promise{Amount: big.NewInt(1), Fee: big.NewInt(1), ChainID: 1},
		R:           "some r",
		AgreementID: big.NewInt(123),
	}

	secondPromise := HermesPromise{
		ChannelID:   "2",
		Identity:    id,
		HermesID:    secondHermes,
		Promise:     crypto.Promise{Amount: big.NewInt(2), Fee: big.NewInt(2), ChainID: 1},
		R:           "some other r",
		AgreementID: big.NewInt(1234),
	}

	// check if errors are wrapped correctly
	_, err = hermesStorage.Get(1, "unknown_id")
	assert.Equal(t, ErrNotFound, err)

	promises, err := hermesStorage.List(HermesPromiseFilter{})
	assert.Equal(t, []HermesPromise{}, promises)
	assert.NoError(t, err)

	// store and check that promise is stored correctly
	err = hermesStorage.Store(firstPromise)
	assert.NoError(t, err)

	promise, err := hermesStorage.Get(1, firstPromise.ChannelID)
	assert.NoError(t, err)
	assert.EqualValues(t, firstPromise, promise)

	promises, err = hermesStorage.List(HermesPromiseFilter{
		ChainID: 1,
	})
	assert.Equal(t, []HermesPromise{firstPromise}, promises)
	assert.NoError(t, err)

	// overwrite the promise, check if it is overwritten
	err = hermesStorage.Store(secondPromise)
	assert.NoError(t, err)

	promise, err = hermesStorage.Get(1, secondPromise.ChannelID)
	assert.NoError(t, err)
	assert.EqualValues(t, secondPromise, promise)

	promises, err = hermesStorage.List(HermesPromiseFilter{
		ChainID: 1,
	})
	assert.Equal(t, []HermesPromise{firstPromise, secondPromise}, promises)
	assert.NoError(t, err)

	overwritingPromise := firstPromise
	overwritingPromise.Promise.Amount = big.NewInt(0)
	err = hermesStorage.Store(overwritingPromise)
	assert.Equal(t, err, ErrAttemptToOverwrite)
}

func TestHermesPromiseStorageDelete(t *testing.T) {
	dir, err := os.MkdirTemp("", "hermesPromiseStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	hermesStorage := NewHermesPromiseStorage(bolt)

	id := identity.FromAddress("0x44440954558C5bFA0D4153B0002B1d1E3E3f5Ff5")
	firstHermes := common.HexToAddress("0x000000acc1")

	firstPromise := HermesPromise{
		ChannelID:   "1",
		Identity:    id,
		HermesID:    firstHermes,
		Promise:     crypto.Promise{Amount: big.NewInt(1), Fee: big.NewInt(1), ChainID: 1},
		R:           "some r",
		AgreementID: big.NewInt(123),
	}

	// should error since no such promise
	err = hermesStorage.Delete(firstPromise)
	assert.Error(t, err)

	err = hermesStorage.Store(firstPromise)
	assert.NoError(t, err)

	promise, err := hermesStorage.Get(1, firstPromise.ChannelID)
	assert.NoError(t, err)
	assert.EqualValues(t, firstPromise, promise)

	err = hermesStorage.Delete(firstPromise)
	assert.NoError(t, err)

	_, err = hermesStorage.Get(1, firstPromise.ChannelID)
	assert.Error(t, err)
}
