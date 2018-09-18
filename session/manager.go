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

// IDGenerator defines method for session id generation
type IDGenerator func() ID

// ConfigProvider provides session config for remote client
type ConfigProvider func() (ServiceConfiguration, error)

// SaveCallback stores newly started sessions
type SaveCallback func(Session)

// PromiseProcessor processes promises at provider side.
// Provider checks promises from consumer and signs them also.
// Provider clears promises from consumer.
type PromiseProcessor interface {
	Start() error
	Stop() error
}

// NewManager returns new session manager
func NewManager(idGenerator IDGenerator, configProvider ConfigProvider, saveCallback SaveCallback, promiseProcessor PromiseProcessor) *manager {
	return &manager{
		generateID:       idGenerator,
		provideConfig:    configProvider,
		saveSession:      saveCallback,
		promiseProcessor: promiseProcessor,

		creationLock: sync.Mutex{},
	}
}

// manager knows how to start and provision session
type manager struct {
	generateID       IDGenerator
	provideConfig    ConfigProvider
	saveSession      SaveCallback
	promiseProcessor PromiseProcessor

	creationLock sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *manager) Create(consumerID identity.Identity) (sessionInstance Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	sessionInstance, err = manager.createSession(consumerID)
	if err != nil {
		return
	}

	err = manager.promiseProcessor.Start()
	if err != nil {
		return
	}

	manager.saveSession(sessionInstance)
	return sessionInstance, nil
}

func (manager *manager) createSession(consumerID identity.Identity) (sessionInstance Session, err error) {
	sessionInstance.ID = manager.generateID()
	sessionInstance.ConsumerID = consumerID
	sessionInstance.Config, err = manager.provideConfig()
	return
}
