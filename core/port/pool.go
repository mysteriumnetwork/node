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
	log "github.com/cihub/seelog"
	"math/rand"
	"time"
)

// Pool hands out ports for service use
type Pool struct {
	start, capacity int
	seed            rand.Source
}

// NewPool creates a port pool that will provide ports from range 40000-50000
func NewPool() *Pool {
	return &Pool{
		start:    40000,
		capacity: 10000,
		seed:     rand.NewSource(time.Now().UnixNano()),
	}
}

// Acquire returns an unused port from the pool
func (pool *Pool) Acquire() Port {
	portNumber := pool.randomPortNumberInRange()
	log.Debug("supplying port from pool: ", portNumber)
	return Port{number: portNumber}
}

func (pool *Pool) randomPortNumberInRange() int {
	return pool.start + rand.New(pool.seed).Intn(pool.capacity)
}
