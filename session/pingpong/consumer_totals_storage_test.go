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

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
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
}
