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

package boltdb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
)

type myTestType struct {
	ID int64 `storm:"id"`
}

var (
	bucket = "test"
)

func createMockStorage(t *testing.T) (*Bolt, func(), error) {
	dir := boltdbtest.CreateTempDir(t)
	close := func() {
		boltdbtest.RemoveTempDir(t, dir)
	}
	storage, err := NewStorage(dir)
	if err != nil {
		close()
		return nil, nil, err
	}
	close = func() {
		storage.Close()
		boltdbtest.RemoveTempDir(t, dir)
	}
	return storage, close, nil
}

func Test_StorageGetByID(t *testing.T) {
	storage, close, err := createMockStorage(t)
	assert.Nil(t, err)
	defer close()

	data := myTestType{
		ID: 1,
	}
	err = storage.Store(bucket, &data)
	assert.Nil(t, err)

	var result myTestType
	err = storage.GetOneByField(bucket, "ID", data.ID, &result)
	assert.Nil(t, err)
	assert.Equal(t, data.ID, result.ID)
}

func Test_StorageGetByID_NotFound(t *testing.T) {
	storage, close, err := createMockStorage(t)
	assert.Nil(t, err)
	defer close()

	bucket := "test"

	var result myTestType
	err = storage.GetOneByField(bucket, "ID", "data.ID", &result)
	assert.Equal(t, "not found", err.Error())
}

func Test_GetLastEntryInBucket(t *testing.T) {
	storage, close, err := createMockStorage(t)
	assert.Nil(t, err)
	defer close()

	data1 := myTestType{
		ID: 1,
	}
	err = storage.Store(bucket, &data1)
	assert.Nil(t, err)

	data2 := myTestType{
		ID: 2,
	}
	err = storage.Store(bucket, &data2)
	assert.Nil(t, err)

	var res myTestType
	err = storage.GetLast(bucket, &res)
	assert.Nil(t, err)
	assert.Equal(t, data2.ID, res.ID)
}

func Test_StorageGetLast_ErrsOnEmpty(t *testing.T) {
	storage, close, err := createMockStorage(t)
	assert.Nil(t, err)
	defer close()

	bucket := "test"

	var result myTestType
	err = storage.GetLast(bucket, &result)
	assert.Equal(t, "not found", err.Error())
}
