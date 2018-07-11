/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	callErrorChan := make(chan error, 1)
	cleanupVal := make(chan int, 1)
	completeRequest := make(chan bool, 1)

	cancelable := NewCancelable()

	cancelableRequest := cancelable.
		NewRequest(func() (interface{}, error) {
			<-completeRequest
			return 1, nil
		}).
		Cleanup(func(val interface{}, err error) {
			cleanupVal <- val.(int)
		})

	go func() {
		_, err := cancelableRequest.Call()
		callErrorChan <- err
	}()

	cancelable.Cancel()

	select {
	case err := <-callErrorChan:
		assert.Equal(t, ErrRequestCancelled, err)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "cancelable Call expected to return in 100 milliseconds")
	}

	completeRequest <- true

	select {
	case val := <-cleanupVal:
		assert.Equal(t, 1, val)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "cancelable cleanup expected to be called in 100 milliseconds")
	}
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
