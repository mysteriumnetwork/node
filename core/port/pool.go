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
	"fmt"
	"math/rand"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/utils/random"
)

// Pool hands out ports for service use
type Pool struct {
	start, capacity int
	rand            *rand.Rand
}

// ServicePortSupplier provides port needed to run a service on
type ServicePortSupplier interface {
	Acquire() (Port, error)
	AcquireMultiple(n int) (ports []Port, err error)
}

// NewFixedRangePool creates a fixed size pool from port.Range
func NewFixedRangePool(r Range) *Pool {
	return &Pool{
		start:    r.Start,
		capacity: r.Capacity(),
		rand:     random.NewTimeSeededRand(),
	}
}

// Acquire returns an unused port in pool's range
func (pool *Pool) Acquire() (Port, error) {
	p, err := pool.seekAvailablePort()
	log.Info().Err(err).Msgf("Supplying port %d", p)
	return Port(p), err
}

func (pool *Pool) seekAvailablePort() (int, error) {
	randomOffset := pool.rand.Intn(pool.capacity)
	for i := 0; i < pool.capacity; i++ {
		p := pool.start + (randomOffset+i)%pool.capacity
		available, err := available(p)
		if available || err != nil {
			return p, err
		}
	}
	return 0, errors.New("port pool is exhausted")
}

// AcquireMultiple returns n unused ports from pool's range.
func (pool *Pool) AcquireMultiple(n int) (ports []Port, err error) {
	if n > pool.capacity {
		return nil, fmt.Errorf("requested more ports (%d) than pool capacity (%d)", n, pool.capacity)
	}

	portSet := make(map[Port]struct{})
	for i := 0; i < 10*n && len(portSet) < n; i++ {
		p, err := pool.Acquire()
		if err != nil {
			continue
		}

		portSet[p] = struct{}{}

		if len(portSet) == n {
			for port := range portSet {
				ports = append(ports, port)
			}
			return ports, nil
		}
	}

	return nil, errors.New("too many collisions")
}
