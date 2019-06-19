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

	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
)

// SessionMap defines map of current sessions
type SessionMap interface {
	Add(session.Session)
	Find(session.ID) (session.Session, bool)
	Remove(session.ID)
}

// clientMap extends current sessions with client id metadata from Openvpn
type clientMap struct {
	sessions SessionMap
	// TODO: use clientID to kill OpenVPN session (client-kill {clientID}) when promise processor instructs so
	sessionClientIDs map[session.ID]int
	sessionMapLock   sync.Mutex
}

// NewClientMap creates a new instance of client map
func NewClientMap(sessionMap SessionMap) *clientMap {
	return &clientMap{
		sessions:         sessionMap,
		sessionClientIDs: make(map[session.ID]int),
	}
}

// FindClientSession returns OpenVPN session instance by given session id
func (cm *clientMap) FindClientSession(clientID int, id session.ID) (session.Session, bool, error) {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	sessionInstance, sessionExist := cm.sessions.Find(id)
	if !sessionExist {
		return session.Session{}, false, errors.New("no underlying session exists, possible break-in attempt")
	}

	_, clientIDExist := cm.sessionClientIDs[id]

	return sessionInstance, clientIDExist, nil
}

// UpdateClientSession updates OpenVPN session with clientID, returns false on clientID conflict
func (cm *clientMap) UpdateClientSession(clientID int, id session.ID) bool {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	_, clientIDExist := cm.sessionClientIDs[id]
	if !clientIDExist {
		cm.sessionClientIDs[id] = clientID
	}

	return cm.sessionClientIDs[id] == clientID
}

// GetClientSessions returns the list of sessions for client found in the client map
func (cm *clientMap) GetClientSessions(clientID int) []session.ID {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()
	res := make([]session.ID, 0)

	for k, v := range cm.sessionClientIDs {
		if v == clientID {
			res = append(res, k)
		}
	}
	return res
}

// RemoveSession removes given session from underlying session managers
func (cm *clientMap) RemoveSession(id session.ID) error {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	_, clientIDExist := cm.sessions.Find(id)
	if !clientIDExist {
		return errors.New("no underlying session exists: " + string(id))
	}

	cm.sessions.Remove(id)
	delete(cm.sessionClientIDs, id)
	return nil
}
