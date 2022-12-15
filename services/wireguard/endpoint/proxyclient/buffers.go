/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package proxyclient

import (
	"sync"
)

// BufferPool represent a buffer sync pool
type BufferPool struct {
	sp sync.Pool
}

// NewBufferPool create a new buffer pool of slice with some length and capacity
func NewBufferPool(size int) (pool *BufferPool) {
	return &BufferPool{
		sp: sync.Pool{
			New: func() interface{} {
				return make([]byte, size, size) // buffer don't grow
			},
		},
	}
}

// Get return some buffer from pool or create new one
func (pool *BufferPool) Get() (buffer []byte) {
	return pool.sp.Get().([]byte)
}

// Put get some buffer and place back it to pool
func (pool *BufferPool) Put(buffer []byte) {
	// not necessary to clean buffer
	pool.sp.Put(buffer)
}
