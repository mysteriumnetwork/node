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
	"errors"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"sync"
)

// NewManager returns session manager which maintains a map of session id -> OpenVPN clientID
func NewManager(m session.Manager) *manager {
	return &manager{
		sessionManager:     m,
		sessionClientIDMap: make(map[session.SessionID]int),
		creationLock:       sync.Mutex{},
	}
}

type manager struct {
	sessionManager     session.Manager
	sessionClientIDMap map[session.SessionID]int
	creationLock       sync.Mutex
}

// Create delegates session creation to underlying session.manager
func (manager *manager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return manager.sessionManager.Create(peerID)
}

// FindSession returns OpenVPN session instance by given session id
func (manager *manager) FindSession(clientID int, id session.SessionID) (session.Session, bool, error) {
	sessionInstance, foundSession := manager.sessionManager.FindSession(id)

	if !foundSession {
		return session.Session{}, false, errors.New("no underlying session exists, possible break-in attempt")
	}

	activeClientID, foundClientID := manager.sessionClientIDMap[id]

	if foundClientID && clientID != activeClientID {
		return sessionInstance, false, errors.New("provided clientID does not mach active clientID")
	}

	return sessionInstance, foundClientID, nil
}

// UpdateSession updates OpenVPN session with clientID, returns false on clientID conflict
func (manager *manager) UpdateSession(clientID int, id session.SessionID) bool {
	_, foundClientID := manager.sessionClientIDMap[id]
	if !foundClientID {
		manager.creationLock.Lock()
		defer manager.creationLock.Unlock()
		manager.sessionClientIDMap[id] = clientID
	}

	return manager.sessionClientIDMap[id] == clientID
}

// RemoveSession removes given session from underlying session managers
func (manager *manager) RemoveSession(id session.SessionID) {
	manager.sessionManager.RemoveSession(id)
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	delete(manager.sessionClientIDMap, id)
}
