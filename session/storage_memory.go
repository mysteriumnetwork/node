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
)

// NewStorageMemory initiates new session storage
func NewStorageMemory() *StorageMemory {
	sm := &StorageMemory{
		sessions: make(map[ID]Session),
		lock:     sync.Mutex{},
	}
	return sm
}

// StorageMemory maintains all current sessions in memory
type StorageMemory struct {
	sessions map[ID]Session
	lock     sync.Mutex
}

// Add puts given session to storage. Multiple sessions per peerID is possible in case different services are used
func (storage *StorageMemory) Add(sessionInstance Session) {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	storage.sessions[sessionInstance.ID] = sessionInstance
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

// UpdateDataTransfer updates the data transfer info on the session
func (storage *StorageMemory) UpdateDataTransfer(id ID, up, down int64) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
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
}

// RemoveForService removes all sessions which belong to given service
func (storage *StorageMemory) RemoveForService(serviceID string) {
	sessions := storage.GetAll()
	for _, session := range sessions {
		if session.serviceID == serviceID {
			storage.Remove(session.ID)
		}
	}
}
