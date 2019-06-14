/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package session

import (
	"sync"

	"github.com/mysteriumnetwork/node/session/event"
)

type bus interface {
	Publish(topic string, data interface{})
	SubscribeAsync(topic string, f interface{}) error
}

// NewStorageMemory initiates new session storage
func NewStorageMemory(bus bus) *StorageMemory {
	sm := &StorageMemory{
		sessions: make(map[ID]Session),
		lock:     sync.Mutex{},
		bus:      bus,
	}
	// intentionally ignoring the error here, it will only error in case ConsumeDataTransferedEvent is not a function
	_ = bus.SubscribeAsync(event.DataTransfered, sm.ConsumeDataTransferedEvent)
	return sm
}

// StorageMemory maintains all current sessions in memory
type StorageMemory struct {
	sessions map[ID]Session
	lock     sync.Mutex
	bus      bus
}

// Add puts given session to storage. Multiple sessions per peerID is possible in case different services are used
func (storage *StorageMemory) Add(sessionInstance Session) {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	storage.sessions[sessionInstance.ID] = sessionInstance
	go storage.bus.Publish(event.Topic, event.Payload{
		ID:     string(sessionInstance.ID),
		Action: event.Created,
	})
}

// GetAll returns all sessions in storage
func (storage *StorageMemory) GetAll() []Session {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	// we're never gonna have more than 100000 sessions ongoing on a single node - performance here should not be an issue.
	// see Benchmark_Storage_GetAll
	sessions := make([]Session, len(storage.sessions))

	i := 0
	for _, value := range storage.sessions {
		sessions[i] = value
		i++
	}
	return sessions
}

// Find returns underlying session instance
func (storage *StorageMemory) Find(id ID) (Session, bool) {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	if instance, found := storage.sessions[id]; found {
		if len(storage.sessions) == 1 {
			instance.Last = true
		}
		return instance, true
	}

	return Session{}, false
}

// ConsumeDataTransferedEvent consumes the data transfer event
func (storage *StorageMemory) ConsumeDataTransferedEvent(e event.DataTransferEventPayload) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	// From a server perspective, bytes up are the actual bytes the client downloaded(aka the bytes we pushed to the consumer)
	// To lessen the confusion, I suggest having the bytes reversed on the session instance.
	// This way, the session will show that it downloaded the bytes in a manner that is easier to comprehend.
	storage.updateDataTransfer(ID(e.ID), e.Down, e.Up)
	go storage.bus.Publish(event.Topic, event.Payload{
		ID:     e.ID,
		Action: event.Updated,
	})
}

func (storage *StorageMemory) updateDataTransfer(id ID, up, down int64) {
	if instance, found := storage.sessions[id]; found {
		instance.DataTransfered.Down = down
		instance.DataTransfered.Up = up
		storage.sessions[id] = instance
	}
}

// Remove removes given session from underlying storage
func (storage *StorageMemory) Remove(id ID) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	delete(storage.sessions, id)
	go storage.bus.Publish(event.Topic, event.Payload{
		ID:     string(id),
		Action: event.Removed,
	})
}

// RemoveForService removes all sessions which belong to given service
func (storage *StorageMemory) RemoveForService(serviceId string) {
	sessions := storage.GetAll()
	for _, session := range sessions {
		if session.serviceID == serviceId {
			storage.Remove(session.ID)
		}
	}
}
