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
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/traversal"
	sevent "github.com/mysteriumnetwork/node/session/event"
	"github.com/pkg/errors"
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

// ConfigParams session configuration parameters
type ConfigParams struct {
	SessionServiceConfig   ServiceConfiguration
	SessionDestroyCallback DestroyCallback
	TraversalParams        *traversal.Params
}

type publisher interface {
	Publish(topic string, data interface{})
}

// ConfigNegotiator is able to handle config negotiations
type ConfigNegotiator interface {
	ProvideConfig(sessionConfig json.RawMessage, traversalParams *traversal.Params) (*ConfigParams, error)
}

// ConfigProvider provides session config for remote client
type ConfigProvider func(sessionConfig json.RawMessage, traversalParams *traversal.Params) (*ConfigParams, error)

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
type BalanceTrackerFactory func(consumer, provider, issuer identity.Identity) (PaymentEngine, error)

// PaymentEngineFactory creates a new instance of payment engine
type PaymentEngineFactory func() (PaymentEngine, error)

// NATEventGetter lets us access the last known traversal event
type NATEventGetter interface {
	LastEvent() *event.Event
}

// NewManager returns new session Manager
func NewManager(
	currentProposal market.ServiceProposal,
	idGenerator IDGenerator,
	sessionStorage Storage,
	balanceTrackerFactory BalanceTrackerFactory,
	paymentEngineFactory PaymentEngineFactory,
	natPingerChan func(*traversal.Params),
	natEventGetter NATEventGetter,
	serviceId string,
	publisher publisher,
) *Manager {
	return &Manager{
		currentProposal:       currentProposal,
		generateID:            idGenerator,
		sessionStorage:        sessionStorage,
		balanceTrackerFactory: balanceTrackerFactory,
		natPingerChan:         natPingerChan,
		natEventGetter:        natEventGetter,
		serviceId:             serviceId,
		publisher:             publisher,
		paymentEngineFactory:  paymentEngineFactory,
		creationLock:          sync.Mutex{},
	}
}

// Manager knows how to start and provision session
type Manager struct {
	currentProposal       market.ServiceProposal
	generateID            IDGenerator
	sessionStorage        Storage
	balanceTrackerFactory BalanceTrackerFactory
	paymentEngineFactory  PaymentEngineFactory
	provideConfig         ConfigProvider
	natPingerChan         func(*traversal.Params)
	natEventGetter        NATEventGetter
	serviceId             string
	publisher             publisher

	creationLock sync.Mutex
}

// Create creates session instance. Multiple sessions per peerID is possible in case different services are used
func (manager *Manager) Create(consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int, config ServiceConfiguration, pingerParams *traversal.Params) (sessionInstance Session, err error) {
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
	sessionInstance.ServiceID = manager.serviceId
	sessionInstance.ConsumerID = consumerID
	sessionInstance.done = make(chan struct{})
	sessionInstance.Config = config
	sessionInstance.CreatedAt = time.Now().UTC()

	// TODO: this whole block needs to go when we deprecate the old payment pingpong
	var paymentEngine PaymentEngine
	if consumerInfo.Supports == PaymentVersionV2 {
		log.Info("using new payments")
		engine, err := manager.paymentEngineFactory()
		if err != nil {
			return sessionInstance, err
		}
		paymentEngine = engine
	} else {
		log.Info("using legacy payments")
		balanceTracker, err := manager.balanceTrackerFactory(consumerID, identity.FromAddress(manager.currentProposal.ProviderID), consumerInfo.IssuerID)
		if err != nil {
			return sessionInstance, err
		}
		paymentEngine = balanceTracker
	}

	// stop the balance tracker once the session is finished
	go func() {
		<-sessionInstance.done
		close(pingerParams.Cancel)
		paymentEngine.Stop()
	}()

	go func() {
		err := paymentEngine.Start()
		if err != nil {
			log.Error(managerLogPrefix, "payment engine error: ", err)
			destroyErr := manager.Destroy(consumerID, string(sessionInstance.ID))
			if destroyErr != nil {
				log.Error(managerLogPrefix, "session cleanup failed: ", err)
			}
		}
	}()

	manager.natPingerChan(pingerParams)
	manager.sessionStorage.Add(sessionInstance)
	return sessionInstance, nil
}

// Acknowledge marks the session as successfuly established as far as the consumer is concerned.
func (manager *Manager) Acknowledge(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	sessionInstance, found := manager.sessionStorage.Find(ID(sessionID))

	if !found {
		return ErrorSessionNotExists
	}

	if sessionInstance.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.publisher.Publish(sevent.Topic, sevent.Payload{
		Action: sevent.Acknowledged,
		ID:     sessionID,
	})

	return nil
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
	close(sessionInstance.done)

	return nil
}
