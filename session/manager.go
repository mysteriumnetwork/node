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
	"errors"
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	discovery_dto "github.com/mysteriumnetwork/node/service_discovery/dto"
)

var (
	// ErrorInvalidProposal is validation error then invalid proposal requested for session creation
	ErrorInvalidProposal = errors.New("proposal does not exist")
)

// IDGenerator defines method for session id generation
type IDGenerator func() (ID, error)

// ConfigProvider provides session config for remote client
type ConfigProvider func() (ServiceConfiguration, error)

// SaveCallback stores newly started sessions
type SaveCallback func(Session)

// PromiseProcessor processes promises at provider side.
// Provider checks promises from consumer and signs them also.
// Provider clears promises from consumer.
type PromiseProcessor interface {
	Start(discovery_dto.ServiceProposal) error
	Stop() error
}

// Manager defines methods for session management
type Manager interface {
	Create(consumerID identity.Identity, proposalID int) (Session, error)
	Destroy(consumerID identity.Identity, sessionID string) error
}

// NewManager returns new session manager
func NewManager(
	currentProposal discovery_dto.ServiceProposal,
	idGenerator IDGenerator,
	configProvider ConfigProvider,
	sessionStorage *StorageMemory,
	promiseProcessor PromiseProcessor,
) *manager {
	return &manager{
		currentProposal:  currentProposal,
		generateID:       idGenerator,
		provideConfig:    configProvider,
		sessionStorage:   sessionStorage,
		promiseProcessor: promiseProcessor,

		creationLock: sync.Mutex{},
	}
}

// manager knows how to start and provision session
type manager struct {
	currentProposal  discovery_dto.ServiceProposal
	generateID       IDGenerator
	provideConfig    ConfigProvider
	sessionStorage   *StorageMemory
	promiseProcessor PromiseProcessor

	creationLock sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *manager) Create(consumerID identity.Identity, proposalID int) (sessionInstance Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	if manager.currentProposal.ID != proposalID {
		err = ErrorInvalidProposal
		return
	}

	sessionInstance, err = manager.createSession(consumerID)
	if err != nil {
		return
	}

	err = manager.promiseProcessor.Start(manager.currentProposal)
	if err != nil {
		return
	}

	manager.sessionStorage.Add(sessionInstance)
	return sessionInstance, nil
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *manager) Destroy(consumerID identity.Identity, sessionID string) (err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	if manager.currentProposal.ID != proposalID {
		err = ErrorInvalidProposal
		return
	}

	err = manager.promiseProcessor.Stop()
	if err != nil {
		return
	}

	sessionInstance, err = manager.sessionStorage.Remove(sessionID)
	if err != nil {
		return
	}

	return nil
}

func (manager *manager) createSession(consumerID identity.Identity) (sessionInstance Session, err error) {
	sessionInstance.ID, err = manager.generateID()
	if err != nil {
		return
	}
	sessionInstance.ConsumerID = consumerID
	sessionInstance.Config, err = manager.provideConfig()
	return
}
