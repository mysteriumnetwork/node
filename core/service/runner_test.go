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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockRunnable struct {
	killErr  error
	waitErr  error
	startErr error
	wg       sync.WaitGroup
}

func (mr *MockRunnable) Start(options Options) (err error) {
	mr.wg.Add(1)
	return mr.startErr
}

func (mr *MockRunnable) Wait() error {
	mr.wg.Wait()
	return mr.waitErr
}

func (mr *MockRunnable) Kill() error {
	mr.wg.Done()
	return mr.killErr
}

type mockFactory struct {
	MockRunnable *MockRunnable
}

func (mf *mockFactory) serviceFactory() RunnableService {
	return mf.MockRunnable
}

func Test_RunnerErrsOnNonExistantService(t *testing.T) {
	m := &mockFactory{}
	c := make(chan error, 1)
	runner := NewRunner(m.serviceFactory)
	sType := "service"
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("unknown service type %q", sType), err.Error())
}

func Test_RunnerErrsOnStart(t *testing.T) {
	fakeErr := errors.New("error")

	m := &mockFactory{MockRunnable: &MockRunnable{
		startErr: fakeErr,
	}}
	c := make(chan error, 1)
	sType := "test"

	runner := NewRunner(m.serviceFactory)
	runner.Register(sType)
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Error(t, err)
	assert.Equal(t, fakeErr, err)
}

func Test_RunnerBubblesErrors(t *testing.T) {
	fakeErr := errors.New("error")
	m := &mockFactory{MockRunnable: &MockRunnable{
		waitErr: fakeErr,
	}}
	c := make(chan error, 1)
	sType := "test"

	runner := NewRunner(m.serviceFactory)
	runner.Register(sType)
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Nil(t, err)

	errs := runner.KillAll()
	assert.Len(t, errs, 0)

	assert.Equal(t, fakeErr, <-c)
}

func Test_RunnerKillReturnsErrors(t *testing.T) {
	fakeErr := errors.New("error")
	m := &mockFactory{MockRunnable: &MockRunnable{
		killErr: fakeErr,
	}}
	c := make(chan error, 1)
	sType := "test"

	runner := NewRunner(m.serviceFactory)
	runner.Register(sType)

	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Nil(t, err)

	errs := runner.KillAll()
	assert.Len(t, errs, 1)
	assert.Equal(t, fakeErr, errs[0])
}
