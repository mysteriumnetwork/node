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

type clientMap struct {
	sessionManager   session.Manager
	sessionClientIDs map[session.SessionID]int
	sessionMapLock   sync.Mutex
}

// FindClientSession returns OpenVPN session instance by given session id
func (cm *clientMap) FindClientSession(clientID int, id session.SessionID) (session.Session, bool, error) {
	sessionInstance, foundSession := cm.sessionManager.FindSession(id)

	if !foundSession {
		return session.Session{}, false, errors.New("no underlying session exists, possible break-in attempt")
	}

	sessionClientID, clientIDFound := cm.sessionClientIDs[id]

	if clientIDFound && clientID != sessionClientID {
		return sessionInstance, false, errors.New("provided clientID does not mach active clientID")
	}

	return sessionInstance, clientIDFound, nil
}

// UpdateClientSession updates OpenVPN session with clientID, returns false on clientID conflict
func (cm *clientMap) UpdateClientSession(clientID int, id session.SessionID) bool {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	_, foundClientID := cm.sessionClientIDs[id]
	if !foundClientID {
		cm.sessionClientIDs[id] = clientID
	}

	return cm.sessionClientIDs[id] == clientID
}

// RemoveSession removes given session from underlying session managers
func (cm *clientMap) RemoveSession(id session.SessionID) error {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	_, found := cm.sessionManager.FindSession(id)
	if !found {
		return errors.New("no underlying session exists: " + string(id))
	}

	cm.sessionManager.RemoveSession(id)
	delete(cm.sessionClientIDs, id)
	return nil
}
