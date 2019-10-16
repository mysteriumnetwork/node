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

package transactor

import (
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTopUpper_RetriesNTimes(t *testing.T) {
	mt := &mockTransactor{
		errToReturn: errors.New("explosions everywhere"),
	}

	tu := NewTopUpper(mt, 3, time.Nanosecond)
	tu.handleRegistrationEvent(identity.FromAddress("id"))

	assert.Equal(t, 3, mt.getCallCount())
}

func TestTopUpper_AbortsRetriesOnStop(t *testing.T) {
	mt := &mockTransactor{
		errToReturn: errors.New("explosions everywhere"),
	}

	tu := NewTopUpper(mt, 3, time.Second)
	done := make(chan struct{})
	go func() {
		tu.handleRegistrationEvent(identity.FromAddress("id"))
		done <- struct{}{}
	}()
	time.Sleep(time.Millisecond * 10)
	tu.Stop()
	<-done

	assert.Equal(t, 1, mt.getCallCount())
}

type mockTransactor struct {
	errToReturn error
	timesCalled int
	lock        sync.Mutex
}

func (mt *mockTransactor) getCallCount() int {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	return mt.timesCalled
}
func (mt *mockTransactor) incrementCallCount() {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	mt.timesCalled++
}

func (mt *mockTransactor) TopUp(id string) error {
	mt.incrementCallCount()
	return mt.errToReturn
}
