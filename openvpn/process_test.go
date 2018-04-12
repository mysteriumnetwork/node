package openvpn

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestWaitAndStopProcessDoesNotDeadLocks(t *testing.T) {
	process := NewProcess("testdata/infinite-loop.sh", "[process-log] ")
	processStarted := sync.WaitGroup{}
	processStarted.Add(1)

	processWaitExited := make(chan int, 1)
	processStopExited := make(chan int, 1)

	go func() {
		assert.NoError(t, process.Start([]string{}))
		processStarted.Done()
		process.Wait()
		processWaitExited <- 1
	}()
	processStarted.Wait()

	go func() {
		process.Stop()
		processStopExited <- 1
	}()

	select {
	case <-processWaitExited:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Process.Wait() didn't return in 100 miliseconds")
	}

	select {
	case <-processStopExited:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Process.Stop() didn't return in 100 miliseconds")
	}
}

func TestWaitReturnsIfProcessDies(t *testing.T) {
	process := NewProcess("testdata/100-milisec-process.sh", "[process-log] ")
	processWaitExited := make(chan int, 1)

	go func() {
		process.Wait()
		processWaitExited <- 1
	}()

	assert.NoError(t, process.Start([]string{}))
	select {
	case <-processWaitExited:
	case <-time.After(500 * time.Millisecond):
		assert.Fail(t, "Process.Wait() didn't return on time")
	}
}
