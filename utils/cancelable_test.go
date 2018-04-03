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
		Request(func() (interface{}, error) {
			return 1, errors.New("message")
		}).
		Call()

	assert.Equal(t, 1, val.(int))
	assert.Equal(t, errors.New("message"), err)
}

func TestBlockingFunctionIsCancelledIfCancelWasCalled(t *testing.T) {
	cancelable := NewCancelable().
		Request(func() (interface{}, error) {
			return nil, nil
		})
	cancelable.Cancel()
	_, err := cancelable.Call()

	assert.Equal(t, ErrRequestCancelled, err)
}

func TestCleanupFunctionIsCalledWithReturnedValueIfCancelWasCalled(t *testing.T) {
	var cleanupVal int
	cleanupWaiter := sync.WaitGroup{}
	cleanupWaiter.Add(1)

	cancelable := NewCancelable().
		Request(func() (interface{}, error) {
			return 1, nil
		}).
		Cleanup(func(val interface{}, err error) {
			cleanupVal = val.(int)
			cleanupWaiter.Done()
		})

	cancelable.Cancel()
	_, err := cancelable.Call()
	cleanupWaiter.Wait()

	assert.Equal(t, ErrRequestCancelled, err)
	assert.Equal(t, 1, cleanupVal)
}

func TestCleanupFunctionIsNotCalledIfBlockingFunctionReturnsError(t *testing.T) {
	var cleanupCalled = false
	cancelable := NewCancelable().
		Request(func() (interface{}, error) {
			return 5, errors.New("failed")
		}).
		Cleanup(func(val interface{}, err error) {
			cleanupCalled = true
		})
	cancelable.Cancel()
	_, err := cancelable.Call()
	assert.Equal(t, ErrRequestCancelled, err)
	assert.False(t, cleanupCalled)

}

func TestRealBlockingFunctionIsCancelled(t *testing.T) {
	errorChannel := make(chan error, 1)
	cancelable := NewCancelable().
		Request(func() (interface{}, error) {
			select {} //effective infinite loop - blocks forever
			return 1, nil
		})

	go func() {
		_, err := cancelable.Call()
		errorChannel <- err
	}()

	cancelable.Cancel()
	select {
	case err := <-errorChannel:
		assert.Equal(t, ErrRequestCancelled, err)
	case <-time.After(300 * time.Millisecond):
		assert.Fail(t, "Timed out while waiting for CancelableAction to produce error")
	}
}

func TestUnspecifiedActionMethodProducesError(t *testing.T) {
	_, err := NewCancelable().Call()
	assert.Equal(t, ErrUndefinedRequest, err)
}

func TestSkipOnErrorProvidesFunctionWhichIsCalledOnlyWhenErrorParameterIsNil(t *testing.T) {
	called := false
	testFunction := SkipOnError(func(interface{}) {
		called = true
	})
	testFunction(1, nil)
	assert.True(t, called)

	called = false
	testFunction(1, errors.New("big error"))
	assert.False(t, called)
}
