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

// EventBusEntry represents the entry in publisher's history
type EventBusEntry struct {
	Topic string
	Event interface{}
}

// EventBus is a fake event bus.
type EventBus struct {
	publishLast    interface{}
	publishHistory []EventBusEntry
	lock           sync.Mutex
}

// NewEventBus creates a new fake event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		publishHistory: make([]EventBusEntry, 0),
	}
}

// Publish fakes publish.
func (mp *EventBus) Publish(topic string, event interface{}) {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	mp.publishHistory = append(mp.publishHistory, EventBusEntry{
		Topic: topic,
		Event: event,
	})
	mp.publishLast = event
}

// SubscribeAsync fakes async subsribe.
func (mp *EventBus) SubscribeAsync(topic string, fn interface{}) error {
	return nil
}

// Subscribe fakes subscribe.
func (mp *EventBus) Subscribe(topic string, fn interface{}) error {
	return nil
}

// SubscribeWithUID fakes subscribe.
func (mp *EventBus) SubscribeWithUID(topic, uid string, fn interface{}) error {
	return nil
}

// Unsubscribe fakes unsubscribe.
func (mp *EventBus) Unsubscribe(topic string, fn interface{}) error {
	return nil
}

// UnsubscribeWithUID fakes unsubscribe.
func (mp *EventBus) UnsubscribeWithUID(topic, uid string, fn interface{}) error {
	return nil
}

// Pop pops the last event for assertions.
func (mp *EventBus) Pop() interface{} {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	result := mp.publishLast
	mp.publishLast = nil
	return result
}

// Clear clears the event history
func (mp *EventBus) Clear() {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	mp.publishHistory = make([]EventBusEntry, 0)
}

// GetEventHistory fetches the event history
func (mp *EventBus) GetEventHistory() []EventBusEntry {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	return mp.publishHistory
}
