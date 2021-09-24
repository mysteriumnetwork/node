/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package port

import (
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testIterations = 10_000

func TestAcquiredPortsAreUsable(t *testing.T) {
	pool := NewFixedRangePool(Range{10000, 60000})

	port, _ := pool.Acquire()
	err := listenUDP(port.Num())

	assert.NoError(t, err)
}

func iteratedTest(t *testing.T) {
	start := 59980
	end := 60000
	pool := NewFixedRangePool(Range{start, end})

	for i := 0; i < testIterations; i++ {
		port, err := pool.Acquire()
		if err != nil {
			t.Errorf("Failed to acquire port: %v", err)
			return
		}
		if port.Num() >= end || port.Num() < start {
			t.Errorf("Port number %d doesn't fits range %d:%d", port, start, end)
			return
		}
	}
}

func TestFitsPoolRange(t *testing.T) {
	iteratedTest(t)
}

func TestConcurrentUsage(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			iteratedTest(t)
		}()
	}
	wg.Wait()
}

func listenUDP(port int) error {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer udpConn.Close()
	return nil
}
