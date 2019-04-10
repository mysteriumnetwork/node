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
	rng             *rand.Rand
}

// NewPool creates a port pool that will provide ports from range 40000-50000
func NewPool() *Pool {
	return &Pool{
		start:    40000,
		capacity: 10000,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Acquire returns an unused port in pool's range
func (pool *Pool) Acquire(protocol string) (Port, error) {
	p := pool.randomPort()
	if !available(protocol, p) {
		var err error
		p, err = pool.seekAvailablePort(protocol)
		if err != nil {
			return 0, err
		}
	}
	log.Debugf("port pool: supplying %s port %d", protocol, p)
	return Port(p), nil
}

func (pool *Pool) randomPort() int {
	return pool.start + pool.rng.Intn(pool.capacity)
}

func (pool *Pool) seekAvailablePort(protocol string) (int, error) {
	for i := 0; i < pool.capacity; i++ {
		p := pool.start + i
		if available(protocol, p) {
			return p, nil
		}
	}
	return 0, errors.New("port pool is exhausted")
}
