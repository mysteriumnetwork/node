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
	"sync"

	"github.com/mysteriumnetwork/node/session"
)

// SessionMap defines map of current sessions
type SessionMap interface {
	Add(session.Session)
	Find(session.ID) (session.Session, bool)
	Remove(session.ID)
}

// clientMap extends current sessions with client id metadata from Openvpn
type clientMap struct {
	sessions         SessionMap
	sessionClientIDs map[session.ID]int
	sessionMapLock   sync.Mutex
}

// FindClientSession returns OpenVPN session instance by given session id
func (cm *clientMap) FindClientSession(clientID int, id session.ID) (session.Session, bool, error) {
	sessionInstance, sessionExist := cm.sessions.Find(id)
	if !sessionExist {
		return session.Session{}, false, errors.New("no underlying session exists, possible break-in attempt")
	}

	sessionClientID, clientIDExist := cm.sessionClientIDs[id]
	if clientIDExist && clientID != sessionClientID {
		return sessionInstance, false, errors.New("provided clientID does not mach active clientID")
	}

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
