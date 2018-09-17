/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/identity"
)

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(serviceConfigProvider ServiceConfigProvider, idGenerator Generator) *manager {
	return &manager{
		idGenerator:    idGenerator,
		configProvider: serviceConfigProvider,
		sessionMap:     make(map[SessionID]Session),
		creationLock:   sync.Mutex{},
	}
}

type manager struct {
	idGenerator    Generator
	configProvider ServiceConfigProvider
	sessionMap     map[SessionID]Session
	creationLock   sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *manager) Create(peerID identity.Identity) (sessionInstance Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	sessionInstance.ID = manager.idGenerator.Generate()
	sessionInstance.ConsumerID = peerID
	sessionInstance.Config, err = manager.configProvider.ProvideServiceConfig()
	if err != nil {
		return
	}

	manager.sessionMap[sessionInstance.ID] = sessionInstance
	return sessionInstance, nil
}

// FindSession returns underlying session instance
func (manager *manager) FindSession(id SessionID) (Session, bool) {
	sessionInstance, found := manager.sessionMap[id]
	return sessionInstance, found
}

// RemoveSession removes given session from underlying session manager
func (manager *manager) RemoveSession(id SessionID) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	delete(manager.sessionMap, id)
}
