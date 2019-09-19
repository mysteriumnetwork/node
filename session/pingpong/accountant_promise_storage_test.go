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

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAccountantPromiseStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "accountantPromiseStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	accountantStorage := NewAccountantPromiseStorage(bolt)

	firstAccountant := identity.FromAddress(acc.Address.Hex())
	firstPromise, err := crypto.CreatePromise("channelOne", 1, 1, "lockOne", ks, acc.Address)
	assert.NoError(t, err)

	secondPromise, err := crypto.CreatePromise("channelTwo", 2, 2, "lockTwo", ks, acc.Address)
	assert.NoError(t, err)

	// check if errors are wrapped correctly
	_, err = accountantStorage.Get(firstAccountant)
	assert.Equal(t, ErrNotFound, err)

	// store and check that promise is stored correctly
	err = accountantStorage.Store(firstAccountant, *firstPromise)
	assert.NoError(t, err)

	promise, err := accountantStorage.Get(firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, *firstPromise, promise)

	// overwrite the promise, check if it is overwritten
	err = accountantStorage.Store(firstAccountant, *secondPromise)
	assert.NoError(t, err)

	promise, err = accountantStorage.Get(firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, *secondPromise, promise)

	// store two promises, check if both are gotten correctly
	account2, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(account2, "")
	assert.Nil(t, err)

	secondAccountant := identity.FromAddress(account2.Address.Hex())

	err = accountantStorage.Store(secondAccountant, *firstPromise)
	assert.NoError(t, err)

	promise, err = accountantStorage.Get(firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, *secondPromise, promise)

	promise, err = accountantStorage.Get(secondAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, *firstPromise, promise)
}
