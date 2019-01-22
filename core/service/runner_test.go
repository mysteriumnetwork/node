/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is mree software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the mree Software Foundation, either version 3 of the License, or
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

package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockRunnable struct {
	killErr  error
	startErr error
	wg       sync.WaitGroup
}

func (mr *MockRunnable) Start(options Options) (err error) {
	mr.wg.Add(1)
	return mr.startErr
}

func (mr *MockRunnable) Kill() error {
	mr.wg.Done()
	return mr.killErr
}

func wait() {
	time.Sleep(time.Millisecond * 5)
}

func Test_RunnerKillReturnsErrors(t *testing.T) {
	fakeErr := errors.New("error")
	sType := "test"
	sInstance := &MockRunnable{
		killErr: fakeErr,
	}

	runner := NewRunner()
	runner.Register(sType, sInstance)

	go func() {
		wait()
		errs := runner.KillAll()
		assert.Len(t, errs, 1)
		assert.Equal(t, fakeErr, errs[0])
	}()
	err := runner.StartServiceByType(sType, Options{})
	assert.Nil(t, err)
}
