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
	"encoding/json"
	"errors"
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

var (
	// ErrorInvalidProposal is validation error then invalid proposal requested for session creation
	ErrorInvalidProposal = errors.New("proposal does not exist")
	// ErrorSessionNotExists returned when consumer tries to destroy session that does not exists
	ErrorSessionNotExists = errors.New("session does not exists")
	// ErrorWrongSessionOwner returned when consumer tries to destroy session that does not belongs to him
	ErrorWrongSessionOwner = errors.New("wrong session owner")
)

// IDGenerator defines method for session id generation
type IDGenerator func() (ID, error)

// ConfigNegotiator is able to handle config negotiations
type ConfigNegotiator interface {
	ProvideConfig(consumerKey json.RawMessage) (ServiceConfiguration, DestroyCallback, error)
}

// ConfigProvider provides session config for remote client
type ConfigProvider func(consumerKey json.RawMessage) (ServiceConfiguration, DestroyCallback, error)

// DestroyCallback cleanups session
type DestroyCallback func() error

// PromiseProcessor processes promises at provider side.
// Provider checks promises from consumer and signs them also.
// Provider clears promises from consumer.
type PromiseProcessor interface {
	Start(proposal market.ServiceProposal) error
	Stop() error
}

// Storage interface to session storage
type Storage interface {
	Add(sessionInstance Session)
	Find(id ID) (Session, bool)
	Remove(id ID)
}

// NewManager returns new session Manager
func NewManager(
	currentProposal market.ServiceProposal,
	idGenerator IDGenerator,
	sessionStorage Storage,
	promiseProcessor PromiseProcessor,
) *Manager {
	return &Manager{
		currentProposal:  currentProposal,
		generateID:       idGenerator,
		sessionStorage:   sessionStorage,
		promiseProcessor: promiseProcessor,

		creationLock: sync.Mutex{},
	}
}

// Manager knows how to start and provision session
type Manager struct {
	currentProposal  market.ServiceProposal
	generateID       IDGenerator
	sessionStorage   Storage
	promiseProcessor PromiseProcessor

	creationLock sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *Manager) Create(consumerID identity.Identity, proposalID int, config ServiceConfiguration) (sessionInstance Session, err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	if manager.currentProposal.ID != proposalID {
		err = ErrorInvalidProposal
		return
	}

	sessionInstance.ID, err = manager.generateID()
	if err != nil {
		return
	}
	sessionInstance.ConsumerID = consumerID
	sessionInstance.Stop = make(chan struct{})
	sessionInstance.Config = config

	err = manager.promiseProcessor.Start(manager.currentProposal)
	if err != nil {
		return
	}

	manager.sessionStorage.Add(sessionInstance)
	return sessionInstance, nil
}

// Destroy destroys session by given sessionID
func (manager *Manager) Destroy(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	sessionInstance, found := manager.sessionStorage.Find(ID(sessionID))

	if !found {
		return ErrorSessionNotExists
	}

	if sessionInstance.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	err := manager.promiseProcessor.Stop()
	if err != nil {
		return err
	}

	manager.sessionStorage.Remove(ID(sessionID))
	close(sessionInstance.Stop)

	return nil
}
