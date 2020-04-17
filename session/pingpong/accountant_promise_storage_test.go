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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAccountantPromiseStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "accountantPromiseStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ks := identity.NewKeystoreFilesystem(dir, identity.NewMockKeystore(identity.MockKeys), identity.MockDecryptFunc)
	acc, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(acc, "")
	assert.Nil(t, err)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()

	accountantStorage := NewAccountantPromiseStorage(bolt)

	id := identity.FromAddress("0x44440954558C5bFA0D4153B0002B1d1E3E3f5Ff5")
	firstAccountant := acc.Address
	fp, err := crypto.CreatePromise("0x30960954558C5bFA0D4153B0002B1d1E3E3f5Ff5", 1, 1, "0xD87C7cF5FF5FDb85988c9AFEf52Ce00A7112eC2e", ks, acc.Address)
	assert.NoError(t, err)

	firstPromise := AccountantPromise{
		Promise:     *fp,
		R:           "some r",
		AgreementID: 123,
	}

	sp, err := crypto.CreatePromise("0x60d99B9a5Dc8E35aD8f2B9199470008AEeA6db90", 2, 2, "0xbDA8709DA6F7B2B99B7729136dE2fD11aB1bB536", ks, acc.Address)
	assert.NoError(t, err)
	secondPromise := AccountantPromise{
		Promise:     *sp,
		R:           "some other r",
		AgreementID: 1234,
	}

	// check if errors are wrapped correctly
	_, err = accountantStorage.Get(id, firstAccountant)
	assert.Equal(t, ErrNotFound, err)

	// store and check that promise is stored correctly
	err = accountantStorage.Store(id, firstAccountant, firstPromise)
	assert.NoError(t, err)

	promise, err := accountantStorage.Get(id, firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, firstPromise, promise)

	// overwrite the promise, check if it is overwritten
	err = accountantStorage.Store(id, firstAccountant, secondPromise)
	assert.NoError(t, err)

	promise, err = accountantStorage.Get(id, firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, secondPromise, promise)

	// store two promises, check if both are gotten correctly
	account2, err := ks.NewAccount("")
	assert.Nil(t, err)

	err = ks.Unlock(account2, "")
	assert.Nil(t, err)

	secondAccountant := account2.Address

	err = accountantStorage.Store(id, secondAccountant, firstPromise)
	assert.NoError(t, err)

	promise, err = accountantStorage.Get(id, firstAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, secondPromise, promise)

	promise, err = accountantStorage.Get(id, secondAccountant)
	assert.NoError(t, err)
	assert.EqualValues(t, firstPromise, promise)
}
