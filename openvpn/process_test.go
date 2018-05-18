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
