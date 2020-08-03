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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
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
type IDGenerator func() (session.ID, error)

// ConfigParams session configuration parameters
type ConfigParams struct {
	SessionServiceConfig   ServiceConfiguration
	SessionDestroyCallback DestroyCallback
}

// ServiceConfiguration defines service configuration from underlying transport mechanism to be passed to remote party
// should be serializable to json format.
type ServiceConfiguration interface{}

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
			SendInterval:    14 * time.Second,
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
	GetAll() []Session
	Find(id session.ID) (Session, bool)
	Remove(id session.ID)
}

// PaymentEngineFactory creates a new instance of payment engine
type PaymentEngineFactory func(providerID, consumerID identity.Identity, accountantID common.Address, sessionID string) (PaymentEngine, error)

// PaymentEngine is responsible for interacting with the consumer in regard to payments.
type PaymentEngine interface {
	Start() error
	WaitFirstInvoice(time.Duration) error
	Stop()
}

// NATEventGetter lets us access the last known traversal event
type NATEventGetter interface {
	LastEvent() *event.Event
}

// NewSessionManager returns new session SessionManager
func NewSessionManager(
	service *Instance,
	sessionStorage Storage,
	paymentEngineFactory PaymentEngineFactory,
	natEventGetter NATEventGetter,
	publisher publisher,
	channel p2p.Channel,
	config Config,
) *SessionManager {
	return &SessionManager{
		service:              service,
		sessionStorage:       sessionStorage,
		natEventGetter:       natEventGetter,
		publisher:            publisher,
		paymentEngineFactory: paymentEngineFactory,
		channel:              channel,
		config:               config,
	}
}

// SessionManager knows how to start and provision session
type SessionManager struct {
	service              *Instance
	sessionStorage       Storage
	paymentEngineFactory PaymentEngineFactory
	natEventGetter       NATEventGetter
	publisher            publisher
	channel              p2p.Channel
	config               Config
}

// Start starts a session on the provider side for the given consumer.
// Multiple sessions per peerID is possible in case different services are used
func (manager *SessionManager) Start(request *pb.SessionRequest) (_ pb.SessionResponse, err error) {
	session, err := NewSession(manager.service, request)
	if err != nil {
		return pb.SessionResponse{}, errors.Wrap(err, "cannot create new session")
	}
	defer func() {
		if err != nil {
			log.Err(err).Msg("Session failed, disconnecting")
			session.Close()
		}
	}()

	trace := session.tracer.StartStage("Provider whole session create")
	defer func() {
		session.tracer.EndStage(trace)
		traceResult := session.tracer.Finish(manager.publisher, string(session.ID))
		log.Debug().Msgf("Provider connection trace: %s", traceResult)
	}()

	if err = manager.startSession(session); err != nil {
		return pb.SessionResponse{}, err
	}
	if err = manager.paymentLoop(session); err != nil {
		return pb.SessionResponse{}, err
	}

	return manager.providerService(session, manager.channel)
}

// Acknowledge marks the session as successfully established as far as the consumer is concerned.
func (manager *SessionManager) Acknowledge(consumerID identity.Identity, sessionID string) error {
	session, found := manager.sessionStorage.Find(session.ID(sessionID))
	if !found {
		return ErrorSessionNotExists
	}
	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.publisher.Publish(sevent.AppTopicSession, session.toEvent(sevent.AcknowledgedStatus))
	return nil
}

func (manager *SessionManager) startSession(session *Session) error {
	trace := session.tracer.StartStage("Provider session start")
	defer session.tracer.EndStage(trace)

	if err := manager.validateSession(session); err != nil {
		return err
	}

	manager.clearStaleSession(session.ConsumerID, manager.service.Type)

	manager.sessionStorage.Add(*session)
	go func() {
		<-session.Done()
		manager.sessionStorage.Remove(session.ID)
	}()

	go manager.keepAliveLoop(session, manager.channel)

	return nil
}

func (manager *SessionManager) validateSession(session *Session) error {
	if manager.service.Proposal.ID != int(session.request.GetProposalID()) {
		return ErrorInvalidProposal
	}

	if !manager.service.Policies().IsIdentityAllowed(session.ConsumerID) {
		return fmt.Errorf("consumer identity is not allowed: %s", session.ConsumerID.Address)
	}

	return nil
}

func (manager *SessionManager) clearStaleSession(consumerID identity.Identity, serviceType string) {
	// Reading stale session before starting the clean up in goroutine.
	// This is required to make sure we are not cleaning the newly created session.
	for _, session := range manager.sessionStorage.GetAll() {
		if consumerID != session.ConsumerID {
			continue
		}
		if serviceType != session.Proposal.ServiceType {
			continue
		}
		log.Info().Msgf("Cleaning stale session %s for %s consumer", session.ID, consumerID.Address)
		go session.Close()
	}
}

// Destroy destroys session by given sessionID
func (manager *SessionManager) Destroy(consumerID identity.Identity, sessionID string) error {
	session, found := manager.sessionStorage.Find(session.ID(sessionID))
	if !found {
		return ErrorSessionNotExists
	}
	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	session.Close()
	return nil
}

func (manager *SessionManager) paymentLoop(session *Session) error {
	trace := session.tracer.StartStage("Provider payments")
	defer session.tracer.EndStage(trace)

	log.Info().Msg("Using new payments")
	engine, err := manager.paymentEngineFactory(manager.service.ProviderID, session.ConsumerID, session.AccountantID, string(session.ID))
	if err != nil {
		return err
	}

	// stop the balance tracker once the session is finished
	go func() {
		<-session.Done()
		engine.Stop()
	}()

	go func() {
		err := engine.Start()
		if err != nil {
			log.Error().Err(err).Msg("Payment engine error")
			session.Close()
		}
	}()

	log.Info().Msg("Waiting for a first invoice to be paid")
	if err := engine.WaitFirstInvoice(30 * time.Second); err != nil {
		return fmt.Errorf("first invoice was not paid: %w", err)
	}

	return nil
}

func (manager *SessionManager) providerService(session *Session, channel p2p.Channel) (pb.SessionResponse, error) {
	trace := session.tracer.StartStage("Provider config")
	defer session.tracer.EndStage(trace)

	config, err := manager.service.Service().ProvideConfig(string(session.ID), session.request.GetConfig(), channel.ServiceConn())
	if err != nil {
		return pb.SessionResponse{}, fmt.Errorf("cannot get provider config for session %s: %w", string(session.ID), err)
	}

	if config.SessionDestroyCallback != nil {
		go func() {
			<-session.Done()
			config.SessionDestroyCallback()
		}()
	}

	data, err := json.Marshal(config.SessionServiceConfig)
	if err != nil {
		return pb.SessionResponse{}, fmt.Errorf("cannot pack session %s service config: %w", string(session.ID), err)
	}

	return pb.SessionResponse{
		ID:          string(session.ID),
		PaymentInfo: "v3",
		Config:      data,
	}, nil
}

func (manager *SessionManager) keepAliveLoop(sess *Session, channel p2p.Channel) {
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
		case <-sess.Done():
			// Give some time for channel to finish sending last message.
			time.Sleep(10 * time.Second)
			channel.Close()
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

func (manager *SessionManager) sendKeepAlivePing(channel p2p.Channel, sessionID session.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), manager.config.KeepAlive.SendTimeout)
	defer cancel()
	msg := &pb.P2PKeepAlivePing{
		SessionID: string(sessionID),
	}
	_, err := channel.Send(ctx, p2p.TopicKeepAlive, p2p.ProtoMessage(msg))
	return err
}
