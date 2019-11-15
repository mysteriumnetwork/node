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

package registry

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestRegistrationStatusStorage(t *testing.T) {
	dir, err := ioutil.TempDir("", "consumerTotalsStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	defer bolt.Close()
	consumerTotalsStorage := NewRegistrationStatusStorage(bolt)

	mockIdentity := identity.FromAddress("0x001")
	mockStatus := StoredRegistrationStatus{
		RegistrationStatus: RegisteredProvider,
		Identity:           mockIdentity,
	}
	_, err = consumerTotalsStorage.Get(identity.FromAddress("0x001"))
	assert.Equal(t, ErrNotFound, err)

	err = consumerTotalsStorage.Store(mockStatus)
	assert.Nil(t, err)

	res, err := consumerTotalsStorage.Get(mockIdentity)
	assert.Nil(t, err)

	statusesEqual(t, mockStatus, res)

	mockStatus2 := mockStatus
	mockStatus2.RegistrationStatus = RegistrationError
	mockStatus2.Identity = identity.FromAddress("0x002")
	err = consumerTotalsStorage.Store(mockStatus2)
	assert.Nil(t, err)

	all, err := consumerTotalsStorage.GetAll()
	assert.Nil(t, err)
	statusesEqual(t, mockStatus, all[0])
	statusesEqual(t, mockStatus2, all[1])

	// should not update the registered provider status
	err = consumerTotalsStorage.Store(StoredRegistrationStatus{Identity: mockIdentity, RegistrationStatus: RegistrationError})
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockIdentity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredProvider, res.RegistrationStatus)

	// should update status
	err = consumerTotalsStorage.Store(StoredRegistrationStatus{Identity: mockStatus2.Identity, RegistrationStatus: RegisteredConsumer})
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredConsumer, res.RegistrationStatus)

	// should not override registeredConsumer status
	err = consumerTotalsStorage.Store(StoredRegistrationStatus{Identity: mockStatus2.Identity, RegistrationStatus: RegistrationError})
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredConsumer, res.RegistrationStatus)

	// should override the status with RegisteredProvider
	err = consumerTotalsStorage.Store(StoredRegistrationStatus{Identity: mockStatus2.Identity, RegistrationStatus: RegisteredProvider})
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredProvider, res.RegistrationStatus)
}

func statusesEqual(t *testing.T, a, b StoredRegistrationStatus) {
	assert.Equal(t, a.RegistrationStatus, b.RegistrationStatus)
	assert.Equal(t, a.Identity.Address, b.Identity.Address)
	assert.EqualValues(t, a.RegistrationRequest, b.RegistrationRequest)
}
