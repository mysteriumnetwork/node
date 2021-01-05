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

package service

import (
	"sync"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/session"
)

// SessionMap defines map of current sessions
type SessionMap interface {
	Find(session.ID) (*service.Session, bool)
}

// clientMap extends current sessions with client id metadata from Openvpn.
type clientMap struct {
	sessions SessionMap
	// TODO: use clientID to kill OpenVPN session (client-kill {clientID}) when promise processor instructs so
	sessionClientIDs map[int]session.ID
	sessionMapLock   sync.Mutex
}

// NewClientMap creates a new instance of client map.
func NewClientMap(sessionMap SessionMap) *clientMap {
	return &clientMap{
		sessions:         sessionMap,
		sessionClientIDs: make(map[int]session.ID),
	}
}

// Add adds OpenVPN client with used session ID.
func (cm *clientMap) Add(clientID int, sessionID session.ID) {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	cm.sessionClientIDs[clientID] = sessionID
}

// Remove removes given OpenVPN client.
func (cm *clientMap) Remove(clientID int) {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	delete(cm.sessionClientIDs, clientID)
}

// GetSession returns ongoing session instance by given session id.
func (cm *clientMap) GetSession(id session.ID) (*service.Session, bool) {
	return cm.sessions.Find(id)
}

// GetSessionClients returns Openvpn clients which are using given session.
func (cm *clientMap) GetSessionClients(id session.ID) []int {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	res := make([]int, 0)
	for clientID, sessionID := range cm.sessionClientIDs {
		if id == sessionID {
			res = append(res, clientID)
		}
	}
	return res
}

// GetClientSession returns session for given Openvpn client.
func (cm *clientMap) GetClientSession(clientID int) (session.ID, bool) {
	cm.sessionMapLock.Lock()
	defer cm.sessionMapLock.Unlock()

	sessionID, exist := cm.sessionClientIDs[clientID]
	return sessionID, exist
}
