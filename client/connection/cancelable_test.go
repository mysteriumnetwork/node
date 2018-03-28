package connection

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestBlockingFunctionResultIsPropagatedToCaller(t *testing.T) {
	cancel := make(cancelChannel, 1)

	val, err := cancelableAction(func() (interface{}, error) {
		return 1, errors.New("message")
	}, cancel)

	assert.Equal(t, 1, val.(int))
	assert.Equal(t, errors.New("message"), err)
}

func TestBlockingFunctionIsCancelledIfCancelChannelIsClosed(t *testing.T) {
	cancel := make(cancelChannel, 1)
	close(cancel)

	_, err := cancelableAction(func() (interface{}, error) {
		return nil, nil
	}, cancel)

	assert.Equal(t, errActionCancelled, err)
}

func TestCleanupFunctionIsCalledWithReturnedValueIfCancelChannelWasClosed(t *testing.T) {
	cancel := make(cancelChannel, 1)
	close(cancel)

	var cleanupVal int
	cleanupWaiter := sync.WaitGroup{}
	cleanupWaiter.Add(1)
	_, err := cancelableActionWithCleanup(
		func() (interface{}, error) {
			return 1, nil
		},
		func(val interface{}, err error) {
			cleanupVal = val.(int)
			cleanupWaiter.Done()
		},
		cancel,
	)

	cleanupWaiter.Wait()
	assert.Equal(t, errActionCancelled, err)
	assert.Equal(t, 1, cleanupVal)
}

func TestCleanupFunctionIsNotCalledIfBlockingFunctionReturnsError(t *testing.T) {

	cancel := make(cancelChannel, 1)
	close(cancel)

	var cleanupCalled = false
	_, err := cancelableActionWithCleanup(
		func() (interface{}, error) {
			return 5, errors.New("Failed")
		},
		func(val interface{}, err error) {
			cleanupCalled = true
		},
		cancel,
	)

	assert.Equal(t, errActionCancelled, err)
	assert.False(t, cleanupCalled)

}

func TestRealBlockingFunctionIsCancelled(t *testing.T) {
	cancel := make(cancelChannel, 1)
	errorChannel := make(chan error, 1)

	go func() {
		_, err := cancelableAction(func() (interface{}, error) {
			select {} //effective infinite loop - blocks forever
			return 1, nil
		}, cancel)
		errorChannel <- err
	}()

	close(cancel)
	select {
	case err := <-errorChannel:
		assert.Equal(t, errActionCancelled, err)
	case <-time.After(300 * time.Millisecond):
		assert.Fail(t, "Timed out while waiting for cancelableAction to produce error")
	}
}
