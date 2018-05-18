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
	"github.com/mysterium/node/identity"
	"sync"
	"github.com/mysterium/node/session"
)

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(m session.Manager) *manager {
	return &manager{
		sessionManager:     m,
		sessionClientIDMap: make(map[session.SessionID]int),
		creationLock:   	sync.Mutex{},
	}
}

type manager struct {
	sessionManager session.Manager
	sessionClientIDMap     map[session.SessionID]int
	creationLock   sync.Mutex
}

// Create delegates session creation to underlying session.manager
func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return manager.sessionManager.Create(peerID)
}

// FindSession returns session instance by given session id
func (manager *manager) FindSession(id session.SessionID) (session.Session, bool) {
	return manager.sessionManager.FindSession(id)
}

// FindSession finds session and sets clientID if it is not set yet, returns false on clientID conflict
func (manager *manager) FindUpdateSession(clientID int, id session.SessionID) (session.Session, bool) {
	// start enumerating clients from '1', since non-existing key, might return '0' as clientID value
	clientID++
	sessionInstance, found := manager.FindSession(id)
	activeClientID := manager.sessionClientIDMap[id]
	if activeClientID == 0 {
		manager.sessionClientIDMap[id] = clientID
		return sessionInstance, found
	}

	if activeClientID != clientID {
		return session.Session{}, false
	}
	return sessionInstance, found
}

// RemoveSession removes given session from underlying session managers
func (manager *manager) RemoveSession(id session.SessionID) {
	manager.sessionManager.RemoveSession(id)
	delete(manager.sessionClientIDMap, id)
}
