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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	sevent "github.com/mysteriumnetwork/node/session/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

// ConfigParams session configuration parameters
type ConfigParams struct {
	SessionServiceConfig   ServiceConfiguration
	SessionDestroyCallback DestroyCallback
}

type publisher interface {
	Publish(topic string, data interface{})
}

// KeepAliveConfig contains keep alive options.
type KeepAliveConfig struct {
	SendInterval    time.Duration
	SendTimeout     time.Duration
	MaxSendErrCount int
}

// Config contains common configuration options for session manager.
type Config struct {
	KeepAlive KeepAliveConfig
}

// DefaultConfig returns default params.
func DefaultConfig() Config {
	return Config{
		KeepAlive: KeepAliveConfig{
			SendInterval:    3 * time.Minute,
			SendTimeout:     5 * time.Second,
			MaxSendErrCount: 5,
		},
	}
}

// ConfigProvider is able to handle config negotiations
type ConfigProvider interface {
	ProvideConfig(sessionID string, sessionConfig json.RawMessage, conn *net.UDPConn) (*ConfigParams, error)
}

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
	FindBy(opts FindOpts) (ID, bool)
	Remove(id ID)
}

// PaymentEngineFactory creates a new instance of payment engine
type PaymentEngineFactory func(providerID, consumerID identity.Identity, accountantID common.Address, sessionID string) (PaymentEngine, error)

// NATEventGetter lets us access the last known traversal event
type NATEventGetter interface {
	LastEvent() *event.Event
}

// NewManager returns new session Manager
func NewManager(
	currentProposal market.ServiceProposal,
	sessionStorage Storage,
	paymentEngineFactory PaymentEngineFactory,
	natEventGetter NATEventGetter,
	serviceId string,
	publisher publisher,
	channel p2p.Channel,
	config Config,
) *Manager {
	return &Manager{
		currentProposal:      currentProposal,
		sessionStorage:       sessionStorage,
		natEventGetter:       natEventGetter,
		serviceId:            serviceId,
		publisher:            publisher,
		paymentEngineFactory: paymentEngineFactory,
		channel:              channel,
		config:               config,
	}
}

// Manager knows how to start and provision session
type Manager struct {
	currentProposal      market.ServiceProposal
	sessionStorage       Storage
	paymentEngineFactory PaymentEngineFactory
	natEventGetter       NATEventGetter
	serviceId            string
	publisher            publisher
	creationLock         sync.Mutex
	channel              p2p.Channel
	config               Config
}

// Start starts a session on the provider side for the given consumer.
// Multiple sessions per peerID is possible in case different services are used
func (manager *Manager) Start(session *Session, consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int) (err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	if manager.currentProposal.ID != proposalID {
		err = ErrorInvalidProposal
		return
	}

	manager.clearStaleSession(consumerID, manager.currentProposal.ServiceType)

	session.ServiceType = manager.currentProposal.ServiceType
	session.ServiceID = manager.serviceId
	session.ConsumerID = consumerID
	session.done = make(chan struct{})
	session.CreatedAt = time.Now().UTC()

	log.Info().Msg("Using new payments")
	engine, err := manager.paymentEngineFactory(identity.FromAddress(manager.currentProposal.ProviderID), consumerID, common.HexToAddress(consumerInfo.AccountantID.Address), string(session.ID))
	if err != nil {
		return err
	}

	// stop the balance tracker once the session is finished
	go func() {
		<-session.done
		engine.Stop()
	}()

	go func() {
		err := engine.Start()
		if err != nil {
			log.Error().Err(err).Msg("Payment engine error")
			destroyErr := manager.Destroy(consumerID, string(session.ID))
			if destroyErr != nil {
				log.Error().Err(err).Msg("Session cleanup failed")
			}
		}
	}()

	log.Info().Msg("Waiting for a first invoice to be paid")
	// TODO 3 seconds timeout should be increased when all consumers will be able to pay for the first invoice before session start.
	if err := engine.WaitFirstInvoice(3 * time.Second); err != nil {
		return fmt.Errorf("first invoice was not paid: %w", err)
	}

	go manager.keepAliveLoop(session, manager.channel)
	manager.sessionStorage.Add(*session)
	return nil
}

// Acknowledge marks the session as successfully established as far as the consumer is concerned.
func (manager *Manager) Acknowledge(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	session, found := manager.sessionStorage.Find(ID(sessionID))

	if !found {
		return ErrorSessionNotExists
	}

	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.publisher.Publish(sevent.AppTopicSession, sevent.Payload{
		Action: sevent.Acknowledged,
		ID:     sessionID,
	})

	return nil
}

func (manager *Manager) clearStaleSession(consumerID identity.Identity, serviceType string) {
	sessionID, ok := manager.sessionStorage.FindBy(FindOpts{
		Peer:        &consumerID,
		ServiceType: serviceType,
	})
	if ok {
		log.Info().Msgf("Cleaning stale session %s for %s consumer", sessionID, consumerID.Address)
		go manager.Destroy(consumerID, string(sessionID))
	}
}

// Destroy destroys session by given sessionID
func (manager *Manager) Destroy(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	session, found := manager.sessionStorage.Find(ID(sessionID))
	if !found {
		return ErrorSessionNotExists
	}

	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.sessionStorage.Remove(ID(sessionID))
	close(session.done)

	return nil
}

func (manager *Manager) keepAliveLoop(sess *Session, channel p2p.Channel) {
	// TODO: Remove this check once all provider migrates to p2p.
	if channel == nil {
		return
	}

	// Register handler for handling p2p keep alive pings from consumer.
	channel.Handle(p2p.TopicKeepAlive, func(c p2p.Context) error {
		var ping pb.P2PKeepAlivePing
		if err := c.Request().UnmarshalProto(&ping); err != nil {
			return err
		}

		log.Debug().Msgf("Received p2p keepalive ping with SessionID=%s", ping.SessionID)
		return c.OK()
	})

	// Send pings to consumer.
	var errCount int
	for {
		select {
		case <-sess.done:
			return
		case <-time.After(manager.config.KeepAlive.SendInterval):
			if err := manager.sendKeepAlivePing(channel, sess.ID); err != nil {
				log.Err(err).Msgf("Failed to send p2p keepalive ping. SessionID=%s", sess.ID)
				errCount++
				if errCount == manager.config.KeepAlive.MaxSendErrCount {
					log.Error().Msgf("Max p2p keepalive err count reached, closing p2p channel. SessionID=%s", sess.ID)
					channel.Close()
					return
				}
			} else {
				errCount = 0
			}
		}
	}
}

func (manager *Manager) sendKeepAlivePing(channel p2p.Channel, sessionID ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), manager.config.KeepAlive.SendTimeout)
	defer cancel()
	msg := &pb.P2PKeepAlivePing{
		SessionID: string(sessionID),
	}
	_, err := channel.Send(ctx, p2p.TopicKeepAlive, p2p.ProtoMessage(msg))
	return err
}
