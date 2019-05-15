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

// NewFixedPool creates new instance of PoolFixed
func NewPoolFixed(port Port) *PoolFixed {
	return &PoolFixed{
		port:       port,
		randomPool: NewPool(),
	}
}

// PoolFixed hands out a fixed port for service use
type PoolFixed struct {
	port       Port
	randomPool *Pool
}

// Acquire returns an unused port in pool's range
func (pool *PoolFixed) Acquire() (port Port, err error) {
	if pool.port > 0 {
		return pool.port, nil
	}

	port, err = pool.Acquire()
	pool.port = port
	return
}
