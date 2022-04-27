/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var hid = common.HexToAddress("0x8129243802538e426A023FedB0Da20689aee797A")
var rid = common.HexToAddress("0x7129243802538e426A023FedB0Da20689aee797A")
var chainID int64 = 5

func Test_HermesActivityChecker(t *testing.T) {
	t.Run("uses cached value if present and valid", func(t *testing.T) {
		mbc := &mockBc{}
		checker := NewHermesStatusChecker(mbc, nil, time.Minute)
		checker.cachedValues[checker.formKey(chainID, hid)] = HermesStatus{
			IsActive:   true,
			ValidUntil: time.Now().Add(time.Minute),
		}
		status, err := checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)

		assert.Equal(t, 0, mbc.getTimesCalled())
	})

	t.Run("updates value if present and not valid", func(t *testing.T) {
		startTime := time.Now()

		mbc := &mockBc{
			isActiveResult:     true,
			isRegisteredResult: true,
			feeResult:          1,
		}
		checker := NewHermesStatusChecker(mbc, nil, time.Minute)
		checker.cachedValues[checker.formKey(chainID, hid)] = HermesStatus{
			IsActive:   false,
			ValidUntil: time.Now().Add(-time.Minute),
		}
		status, err := checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)

		assert.Equal(t, uint16(1), status.Fee)

		assert.Equal(t, 3, mbc.getTimesCalled())

		v, _ := checker.cachedValues[checker.formKey(chainID, hid)]
		assert.True(t, v.isValid())

		// check if extended by somewhere around a minute
		assert.True(t, v.ValidUntil.After(startTime.Add(time.Minute)))
		assert.False(t, v.ValidUntil.After(startTime.Add(time.Minute).Add(time.Second*2)))
	})

	t.Run("updates and sets cache if not present initially", func(t *testing.T) {
		startTime := time.Now()

		mbc := &mockBc{
			isActiveResult:     true,
			isRegisteredResult: true,
			feeResult:          1,
		}

		checker := NewHermesStatusChecker(mbc, nil, time.Minute)
		status, err := checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)

		assert.Equal(t, uint16(1), status.Fee)

		assert.Equal(t, 3, mbc.getTimesCalled())

		v, _ := checker.cachedValues[checker.formKey(chainID, hid)]
		assert.True(t, v.isValid())

		// check if extended by somewhere around a minute
		assert.True(t, v.ValidUntil.After(startTime.Add(time.Minute)))
		assert.False(t, v.ValidUntil.After(startTime.Add(time.Minute).Add(time.Second*2)))
	})

	t.Run("successive runs do not fetch from source if already cached and valid", func(t *testing.T) {
		mbc := &mockBc{
			isActiveResult:     true,
			isRegisteredResult: true,
			feeResult:          1,
		}

		checker := NewHermesStatusChecker(mbc, nil, time.Minute)
		status, err := checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)

		assert.Equal(t, 3, mbc.getTimesCalled())

		status, err = checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)
		assert.Equal(t, 3, mbc.getTimesCalled())
	})

	t.Run("successive fetch from source if cache invalid", func(t *testing.T) {
		mbc := &mockBc{
			isActiveResult:     true,
			isRegisteredResult: true,
			feeResult:          1,
		}

		checker := NewHermesStatusChecker(mbc, nil, time.Minute)
		status, err := checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)
		assert.Equal(t, 3, mbc.getTimesCalled())

		checker.cachedValues[checker.formKey(chainID, hid)] = HermesStatus{
			IsActive:   true,
			ValidUntil: time.Now().Add(-time.Minute),
		}

		status, err = checker.GetHermesStatus(chainID, rid, hid)
		assert.NoError(t, err)
		assert.True(t, status.IsActive)
		assert.Equal(t, 6, mbc.getTimesCalled())
	})
}

type mockBc struct {
	isActiveResult bool
	isActiveErr    error

	isRegisteredResult bool
	isRegisteredErr    error

	feeResult uint16
	feeErr    error

	timesCalled int
	lock        sync.Mutex
}

func (mbc *mockBc) incTimesCalled() {
	mbc.lock.Lock()
	defer mbc.lock.Unlock()
	mbc.timesCalled++
}

func (mbc *mockBc) getTimesCalled() int {
	mbc.lock.Lock()
	defer mbc.lock.Unlock()
	return mbc.timesCalled
}

func (mbc *mockBc) IsHermesActive(chainID int64, hermesID common.Address) (bool, error) {
	defer mbc.incTimesCalled()
	return mbc.isActiveResult, mbc.isActiveErr
}

func (mbc *mockBc) IsHermesRegistered(chainID int64, registryAddress, hermesID common.Address) (bool, error) {
	defer mbc.incTimesCalled()
	return mbc.isRegisteredResult, mbc.isRegisteredErr
}

func (mbc *mockBc) GetHermesFee(chainID int64, hermesID common.Address) (uint16, error) {
	defer mbc.incTimesCalled()
	return mbc.feeResult, mbc.feeErr
}
