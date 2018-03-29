package connection

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestBlockingFunctionResultIsPropagatedToCaller(t *testing.T) {
	val, err := newCancelable().
		action(func() (interface{}, error) {
			return 1, errors.New("message")
		}).
		call()

	assert.Equal(t, 1, val.(int))
	assert.Equal(t, errors.New("message"), err)
}

func TestBlockingFunctionIsCancelledIfCancelWasCalled(t *testing.T) {
	cancelable := newCancelable().
		action(func() (interface{}, error) {
			return nil, nil
		})
	cancelable.cancel()
	_, err := cancelable.call()

	assert.Equal(t, errActionCancelled, err)
}

func TestCleanupFunctionIsCalledWithReturnedValueIfCancelWasCalled(t *testing.T) {
	var cleanupVal int
	cleanupWaiter := sync.WaitGroup{}
	cleanupWaiter.Add(1)

	cancelable := newCancelable().
		action(func() (interface{}, error) {
			return 1, nil
		}).
		cleanup(func(val interface{}, err error) {
			cleanupVal = val.(int)
			cleanupWaiter.Done()
		})

	cancelable.cancel()
	_, err := cancelable.call()
	cleanupWaiter.Wait()

	assert.Equal(t, errActionCancelled, err)
	assert.Equal(t, 1, cleanupVal)
}

func TestCleanupFunctionIsNotCalledIfBlockingFunctionReturnsError(t *testing.T) {
	var cleanupCalled = false
	cancelable := newCancelable().
		action(func() (interface{}, error) {
			return 5, errors.New("failed")
		}).
		cleanup(func(val interface{}, err error) {
			cleanupCalled = true
		})
	cancelable.cancel()
	_, err := cancelable.call()
	assert.Equal(t, errActionCancelled, err)
	assert.False(t, cleanupCalled)

}

func TestRealBlockingFunctionIsCancelled(t *testing.T) {
	errorChannel := make(chan error, 1)
	cancelable := newCancelable().
		action(func() (interface{}, error) {
			select {} //effective infinite loop - blocks forever
			return 1, nil
		})

	go func() {
		_, err := cancelable.call()
		errorChannel <- err
	}()

	cancelable.cancel()
	select {
	case err := <-errorChannel:
		assert.Equal(t, errActionCancelled, err)
	case <-time.After(300 * time.Millisecond):
		assert.Fail(t, "Timed out while waiting for cancelableAction to produce error")
	}
}

func TestUnspecifiedActionMethodProducesError(t *testing.T) {
	_, err := newCancelable().call()
	assert.Equal(t, errUndefinedAction, err)
}

func TestSkipOnErrorProvidesFunctionWhichIsCalledOnlyWhenErrorParameterIsNil(t *testing.T) {
	called := false
	testFunction := skipOnError(func(interface{}) {
		called = true
	})
	testFunction(1, nil)
	assert.True(t, called)

	called = false
	testFunction(1, errors.New("big error"))
	assert.False(t, called)
}
