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
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	pc "github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestConsumerTotalStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "consumerTotalsStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	consumerTotalsStorage := NewConsumerTotalsStorage(bolt)

	channelAddress := "someAddress"
	var amount uint64 = 12

	// check if errors are wrapped correctly
	_, err = consumerTotalsStorage.Get(channelAddress)
	assert.Equal(t, ErrNotFound, err)

	// store and check that total is stored correctly
	err = consumerTotalsStorage.Store(channelAddress, amount)
	assert.NoError(t, err)

	a, err := consumerTotalsStorage.Get(channelAddress)
	assert.NoError(t, err)
	assert.Equal(t, amount, a)

	var newAmount uint64 = 123
	// overwrite the amount, check if it is overwritten
	err = consumerTotalsStorage.Store(channelAddress, newAmount)
	assert.NoError(t, err)

	a, err = consumerTotalsStorage.Get(channelAddress)
	assert.NoError(t, err)
	assert.EqualValues(t, newAmount, a)

	someOtherChannel := "someOtherChannel"
	// store two amounts, check if both are gotten correctly
	err = consumerTotalsStorage.Store(someOtherChannel, amount)
	assert.NoError(t, err)

	a, err = consumerTotalsStorage.Get(channelAddress)
	assert.NoError(t, err)
	assert.EqualValues(t, newAmount, a)

	a, err = consumerTotalsStorage.Get(someOtherChannel)
	assert.NoError(t, err)
	assert.EqualValues(t, amount, a)

	// test the convenience store method
	pkID, err := crypto.GenerateKey()
	assert.Nil(t, err)
	identity := crypto.PubkeyToAddress(pkID.PublicKey).Hex()

	registryPK, err := crypto.GenerateKey()
	assert.Nil(t, err)
	registry := crypto.PubkeyToAddress(registryPK.PublicKey).Hex()

	channelPK, err := crypto.GenerateKey()
	assert.Nil(t, err)
	channel := crypto.PubkeyToAddress(channelPK.PublicKey).Hex()

	var convenienceAmount uint64 = 11
	cap := ChannelAddressParams{
		Identity:              identity,
		Registry:              registry,
		ChannelImplementation: channel,
	}
	err = consumerTotalsStorage.GenerateAndStore(cap, convenienceAmount)
	assert.Nil(t, err)

	addr, err := pc.GenerateChannelAddress(identity, registry, channel)
	assert.Nil(t, err)

	res, err := consumerTotalsStorage.Get(addr)
	assert.Nil(t, err)

	assert.Equal(t, convenienceAmount, res)
}
