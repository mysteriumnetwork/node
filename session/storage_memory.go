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
	return &StorageMemory{
		sessionMap: make(map[SessionID]Session),
		lock:       sync.Mutex{},
	}
}

// StorageMemory maintains a map of session id -> session
type StorageMemory struct {
	sessionMap map[SessionID]Session
	lock       sync.Mutex
}

// Add puts given session to storage. Multiple sessions per peerID is possible in case different services are used
func (storage *StorageMemory) Add(sessionInstance Session) {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	storage.sessionMap[sessionInstance.ID] = sessionInstance
}

// Find returns underlying session instance
func (storage *StorageMemory) Find(id SessionID) (Session, bool) {
	sessionInstance, found := storage.sessionMap[id]
	return sessionInstance, found
}

// Remove removes given session from underlying storage
func (storage *StorageMemory) Remove(id SessionID) {
	storage.lock.Lock()
	defer storage.lock.Unlock()

	delete(storage.sessionMap, id)
}
