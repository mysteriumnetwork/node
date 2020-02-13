/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package mocks

import (
	"sync"
)

// EventBus is a fake event bus.
type EventBus struct {
	published interface{}
	lock      sync.Mutex
}

// NewEventBus creates a new fake event bus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Publish fakes publish.
func (mp *EventBus) Publish(topic string, event interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	mp.published = event
}

// SubscribeAsync fakes async subsribe.
func (mp *EventBus) SubscribeAsync(topic string, fn interface{}) error {
	return nil
}

// Subscribe fakes subscribe.
func (mp *EventBus) Subscribe(topic string, fn interface{}) error {
	return nil
}

// Pop pops the last event for assertions.
func (mp *EventBus) Pop() interface{} {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	result := mp.published
	mp.published = nil
	return result
}
