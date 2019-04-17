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
	"math/rand"
	"time"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

// Pool hands out ports for service use
type Pool struct {
	start, capacity int
	rand            *rand.Rand
}

// NewPool creates a port pool that will provide ports from range 40000-50000
func NewPool() *Pool {
	return &Pool{
		start:    40000,
		capacity: 10000,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewFixedRangePool creates a fixed size pool from port.Range
func NewFixedRangePool(r Range) *Pool {
	return &Pool{
		start:    r.Start,
		capacity: r.Capacity(),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Acquire returns an unused port in pool's range
func (pool *Pool) Acquire() (port Port, err error) {
	p := pool.randomPort()
	available, err := available(p)
	if err != nil {
		return 0, errors.Wrap(err, "could not acquire port")
	}
	if !available {
		p, err = pool.seekAvailablePort()
	}
	log.Debugf("%s supplying port %v, err %v", logPrefix, p, err)
	return Port(p), errors.Wrap(err, "could not acquire port")
}

func (pool *Pool) randomPort() int {
	return pool.start + pool.rand.Intn(pool.capacity)
}

func (pool *Pool) seekAvailablePort() (int, error) {
	for i := 0; i < pool.capacity; i++ {
		p := pool.start + i
		available, err := available(p)
		if available || err != nil {
			return p, err
		}
	}
	return 0, errors.New("port pool is exhausted")
}
