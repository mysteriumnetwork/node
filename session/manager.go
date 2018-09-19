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

// ServiceConfigProvider defines configuration providing dependency
type ServiceConfigProvider func() (ServiceConfiguration, error)

// IDGenerator defines method for session id generation
type IDGenerator func() SessionID

// SaveCallback stores newly started sessions
type SaveCallback func(Session)

// NewManager returns session manager which maintains a map of session id -> session
func NewManager(idGenerator IDGenerator, configProvider ServiceConfigProvider, saveCallback SaveCallback) *manager {
	return &manager{
		generateID:     idGenerator,
		generateConfig: configProvider,
		saveSession:    saveCallback,
		creationLock:   sync.Mutex{},
	}
}

type manager struct {
	generateID     IDGenerator
	generateConfig ServiceConfigProvider
	saveSession    SaveCallback
	creationLock   sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *manager) Create(peerID identity.Identity) (sessionInstance Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	sessionInstance.ID = manager.generateID()
	sessionInstance.ConsumerID = peerID
	sessionInstance.Config, err = manager.generateConfig()
	if err != nil {
		return
	}

	manager.saveSession(sessionInstance)
	return sessionInstance, nil
}
