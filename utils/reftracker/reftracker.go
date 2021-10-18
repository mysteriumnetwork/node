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

package reftracker

import (
	"errors"
	"sync"
	"time"
)

// DefaultPatrolPeriod is an interval between collect loops
const DefaultPatrolPeriod = 1 * time.Second

var (
	singletonOnce sync.Once
	singletonInst *RefTracker
)

var (
	// ErrNotFound is returned when incremented or decremented element is not
	// found among tracked elements
	ErrNotFound = errors.New("element not found")
)

// Singleton instantiates globally available reference tracker
func Singleton() *RefTracker {
	singletonOnce.Do(func() {
		singletonInst = NewRefTracker(DefaultPatrolPeriod)
	})

	return singletonInst
}

// RefTracker is a collection of elements referenced by string keys.
// Once element reference count hits zero it will be removed after TTL
// unless new references will not appear.
type RefTracker struct {
	elemMux  sync.Mutex
	elements map[string]*trackedElement
	stop     chan struct{}
	stopOnce sync.Once
}

type trackedElement struct {
	lastReleased    time.Time
	ttl             time.Duration
	releaseCallback func()
	refCount        int
}

// NewRefTracker starts RefTracker with
func NewRefTracker(patrolPeriod time.Duration) *RefTracker {
	rt := &RefTracker{
		elements: make(map[string]*trackedElement),
		stop:     make(chan struct{}),
	}
	go rt.patrolLoop(patrolPeriod)
	return rt
}

func (rt *RefTracker) patrolLoop(patrolPeriod time.Duration) {
	for {
		select {
		case <-rt.stop:
			return
		case <-time.After(patrolPeriod):
			now := time.Now()
			var toDelete []string

			rt.elemMux.Lock()
			for key, elem := range rt.elements {
				if elem.refCount == 0 && now.Sub(elem.lastReleased) > elem.ttl {
					toDelete = append(toDelete, key)
					go elem.releaseCallback()
				}
			}

			for _, key := range toDelete {
				delete(rt.elements, key)
			}
			rt.elemMux.Unlock()
		}
	}
}

// Put introduces tracked element along with its TTL and release callback
func (rt *RefTracker) Put(key string, ttl time.Duration, releaseCallback func()) {
	elem := &trackedElement{
		lastReleased:    time.Now(),
		ttl:             ttl,
		releaseCallback: releaseCallback,
		refCount:        0,
	}

	rt.elemMux.Lock()
	defer rt.elemMux.Unlock()

	_, ok := rt.elements[key]
	if !ok {
		rt.elements[key] = elem
	}
}

// Incr increments reference count of an element
func (rt *RefTracker) Incr(key string) error {
	rt.elemMux.Lock()
	defer rt.elemMux.Unlock()

	elem, ok := rt.elements[key]
	if !ok {
		return ErrNotFound
	}

	elem.refCount++

	return nil
}

// Decr decrements reference count of an element
func (rt *RefTracker) Decr(key string) error {
	rt.elemMux.Lock()
	defer rt.elemMux.Unlock()

	elem, ok := rt.elements[key]
	if !ok {
		return ErrNotFound
	}

	elem.refCount--

	if elem.refCount < 0 {
		panic("negative reference count in RefTracker element!")
	}

	if elem.refCount == 0 {
		elem.lastReleased = time.Now()
	}

	return nil
}

// Close stops patrol loop
func (rt *RefTracker) Close() {
	rt.stopOnce.Do(func() {
		close(rt.stop)
	})
}
