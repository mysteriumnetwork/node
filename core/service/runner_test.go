/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type FakeRunnable struct {
	killErr  error
	waitErr  error
	startErr error
	wg       sync.WaitGroup
}

func (fr *FakeRunnable) Start(options Options) (err error) {
	fr.wg.Add(1)
	return fr.startErr
}

func (fr *FakeRunnable) Wait() error {
	fr.wg.Wait()
	return fr.waitErr
}

func (fr *FakeRunnable) Kill() error {
	fr.wg.Done()
	return fr.killErr
}

func Test_RunnerErrsOnNonExistantService(t *testing.T) {
	m := make(map[string]RunnableService)
	c := make(chan error, 1)
	runner := NewRunner(m)
	sType := "service"
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("unknown service type %q", sType), err.Error())
}

func Test_RunnerErrsOnStart(t *testing.T) {
	m := make(map[string]RunnableService)
	c := make(chan error, 1)
	sType := "test"
	fakeErr := errors.New("error")

	m["test"] = &FakeRunnable{
		startErr: fakeErr,
	}

	runner := NewRunner(m)
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Error(t, err)
	assert.Equal(t, fakeErr, err)
}

func Test_RunnerBubblesErrors(t *testing.T) {
	m := make(map[string]RunnableService)
	c := make(chan error, 1)
	sType := "test"
	fakeErr := errors.New("error")

	m["test"] = &FakeRunnable{
		waitErr: fakeErr,
	}

	runner := NewRunner(m)
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Nil(t, err)

	errs := runner.KillAll()
	assert.Len(t, errs, 0)

	assert.Equal(t, fakeErr, <-c)
}

func Test_RunnerKillReturnsErrors(t *testing.T) {
	m := make(map[string]RunnableService)
	c := make(chan error, 1)
	sType := "test"
	fakeErr := errors.New("error")

	m["test"] = &FakeRunnable{
		killErr: fakeErr,
	}

	runner := NewRunner(m)
	err := runner.StartServiceByType(sType, Options{}, c)
	assert.Nil(t, err)

	errs := runner.KillAll()
	assert.Len(t, errs, 1)
	assert.Equal(t, fakeErr, errs[0])
}
