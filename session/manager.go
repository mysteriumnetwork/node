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

	log "github.com/cihub/seelog"
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

const managerLogPrefix = "[session-manager] "

// IDGenerator defines method for session id generation
type IDGenerator func() (ID, error)

// ConfigNegotiator is able to handle config negotiations
type ConfigNegotiator interface {
	ProvideConfig(consumerKey json.RawMessage) (ServiceConfiguration, DestroyCallback, error)
}

// ConfigProvider provides session config for remote client
type ConfigProvider func(consumerKey json.RawMessage) (ServiceConfiguration, DestroyCallback, error)

// DestroyCallback cleanups session
type DestroyCallback func()

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

// BalanceTrackerFactory returns a new instance of balance tracker
type BalanceTrackerFactory func(consumer, provider, issuer identity.Identity) BalanceTracker

// NewManager returns new session Manager
func NewManager(
	currentProposal market.ServiceProposal,
	idGenerator IDGenerator,
	sessionStorage Storage,
	balanceTrackerFactory BalanceTrackerFactory,
) *Manager {
	return &Manager{
		currentProposal:       currentProposal,
		generateID:            idGenerator,
		sessionStorage:        sessionStorage,
		balanceTrackerFactory: balanceTrackerFactory,

		creationLock: sync.Mutex{},
	}
}

// Manager knows how to start and provision session
type Manager struct {
	currentProposal       market.ServiceProposal
	generateID            IDGenerator
	sessionStorage        Storage
	balanceTrackerFactory BalanceTrackerFactory

	creationLock sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *Manager) Create(consumerID identity.Identity, issuerID identity.Identity, proposalID int) (sessionInstance Session, err error) {
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
	sessionInstance.Done = make(chan struct{})

	balanceTracker := manager.balanceTrackerFactory(consumerID, identity.FromAddress(manager.currentProposal.ProviderID), issuerID)

	// stop the balance tracker once the session is finished
	go func() {
		<-sessionInstance.Done
		balanceTracker.Stop()
	}()

	go func() {
		err := balanceTracker.Start()
		if err != nil {
			log.Error(managerLogPrefix, "balance tracker error: ", err)
			destroyErr := manager.Destroy(consumerID, string(sessionInstance.ID))
			if destroyErr != nil {
				log.Error(managerLogPrefix, "session cleanup failed: ", err)
			}
		}
	}()

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

	manager.sessionStorage.Remove(ID(sessionID))
	close(sessionInstance.Done)

	return nil
}
