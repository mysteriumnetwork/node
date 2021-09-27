/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package random

import (
	"math/rand"
	"sync"
)

type concurrentRandomSource struct {
	src rand.Source
	mux sync.Mutex
}

// NewConcurrentRandomSource constructs ConcurrentRandomSource, wrapping
// existing rand.Source/rand.Source64
func NewConcurrentRandomSource(src rand.Source) rand.Source {
	src64, ok := src.(rand.Source64)
	if ok {
		return &concurrentRandomSource64{
			src: src64,
		}
	}
	return &concurrentRandomSource{
		src: src,
	}
}

// Seed is a part of rand.Source interface
func (s *concurrentRandomSource) Seed(seed int64) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.src.Seed(seed)
}

// Int63 is a part of rand.Source interface
func (s *concurrentRandomSource) Int63() int64 {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.src.Int63()
}

type concurrentRandomSource64 struct {
	src rand.Source64
	mux sync.Mutex
}

// Seed is a part of rand.Source interface
func (s *concurrentRandomSource64) Seed(seed int64) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.src.Seed(seed)
}

// Int63 is a part of rand.Source interface
func (s *concurrentRandomSource64) Int63() int64 {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.src.Int63()
}

// Uint64 is a part of rand.Source64 interface
func (s *concurrentRandomSource64) Uint64() uint64 {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.src.Uint64()
}
