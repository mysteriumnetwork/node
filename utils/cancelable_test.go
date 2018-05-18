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

package utils

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestBlockingFunctionResultIsPropagatedToCaller(t *testing.T) {
	val, err := NewCancelable().
		NewRequest(func() (interface{}, error) {
			return 1, errors.New("message")
		}).
		Call()

	assert.Equal(t, 1, val.(int))
	assert.Equal(t, errors.New("message"), err)
}

func TestCleanupFunctionIsCalledWithReturnedValueIfCancelWasCalled(t *testing.T) {
	var cleanupVal int
	cleanupWaiter := sync.WaitGroup{}
	cleanupWaiter.Add(1)

	cancelable := NewCancelable()
	cancelable.Cancel()

	_, err := cancelable.
		NewRequest(func() (interface{}, error) {
			return 1, nil
		}).
		Cleanup(func(val interface{}, err error) {
			cleanupVal = val.(int)
			cleanupWaiter.Done()
		}).
		Call()

	cleanupWaiter.Wait()

	assert.Equal(t, ErrRequestCancelled, err)
	assert.Equal(t, 1, cleanupVal)
}

func TestBlockingFunctionIsCancelled(t *testing.T) {
	errorChannel := make(chan error, 1)
	cancelable := NewCancelable()

	go func() {
		_, err := cancelable.
			NewRequest(func() (interface{}, error) {
				select {} //effective infinite loop - blocks forever
			}).Call()
		errorChannel <- err
	}()

	cancelable.Cancel()
	select {
	case err := <-errorChannel:
		assert.Equal(t, ErrRequestCancelled, err)
	case <-time.After(300 * time.Millisecond):
		assert.Fail(t, "Timed out while waiting for CancelableRequest to produce error")
	}
}

func TestSkipOnErrorProvidesFunctionWhichIsCalledOnlyWhenErrorParameterIsNil(t *testing.T) {
	called := false
	testFunction := InvokeOnSuccess(func(interface{}) {
		called = true
	})
	testFunction(1, nil)
	assert.True(t, called)

	called = false
	testFunction(1, errors.New("big error"))
	assert.False(t, called)
}
