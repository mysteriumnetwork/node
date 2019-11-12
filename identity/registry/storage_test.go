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

func TestNewRegistrationStatusStorage(t *testing.T) {
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
		// TOOD: fill in the registration request
		Identity: mockIdentity,
	}
	_, err = consumerTotalsStorage.Get(identity.FromAddress("0x001"))
	assert.Equal(t, ErrNotFound, err)

	err = consumerTotalsStorage.Store(mockStatus)
	assert.Nil(t, err)

	res, err := consumerTotalsStorage.Get(mockIdentity)
	assert.Nil(t, err)

	assert.EqualValues(t, mockStatus, res)

	mockStatus2 := mockStatus
	mockStatus2.RegistrationStatus = RegistrationError
	mockStatus2.Identity = identity.FromAddress("0x002")
	err = consumerTotalsStorage.Store(mockStatus2)
	assert.Nil(t, err)

	all, err := consumerTotalsStorage.GetAll()
	assert.Nil(t, err)
	assert.EqualValues(t, []StoredRegistrationStatus{mockStatus, mockStatus2}, all)

	// should not update the registered provider status
	err = consumerTotalsStorage.UpdateStatus(mockIdentity, RegistrationError)
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockIdentity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredProvider, res.RegistrationStatus)

	// should update status
	err = consumerTotalsStorage.UpdateStatus(mockStatus2.Identity, RegisteredConsumer)
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredConsumer, res.RegistrationStatus)

	// should not override registeredConsumer status
	err = consumerTotalsStorage.UpdateStatus(mockStatus2.Identity, RegistrationError)
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredConsumer, res.RegistrationStatus)

	// should override the status with RegisteredProvider
	err = consumerTotalsStorage.UpdateStatus(mockStatus2.Identity, RegisteredProvider)
	assert.Nil(t, err)

	res, err = consumerTotalsStorage.Get(mockStatus2.Identity)
	assert.Nil(t, err)
	assert.Equal(t, RegisteredProvider, res.RegistrationStatus)
}
