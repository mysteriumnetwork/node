/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package quality

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/rs/zerolog/log"
)

const (
	appName             = "myst"
	sessionDataName     = "session_data"
	sessionTokensName   = "session_tokens"
	sessionEventName    = "session_event"
	traceEventName      = "trace_event"
	registerIdentity    = "register_identity"
	unlockEventName     = "unlock"
	proposalEventName   = "proposal_event"
	natMappingEventName = "nat_mapping"
)

// Transport allows sending events
type Transport interface {
	SendEvent(Event) error
}

// NewSender creates metrics sender with appropriate transport
func NewSender(transport Transport, appVersion string) *Sender {
	return &Sender{
		Transport:  transport,
		AppVersion: appVersion,

		sessionsActive: make(map[string]sessionContext),
	}
}

// Sender builds events and sends them using given transport
type Sender struct {
	Transport  Transport
	AppVersion string

	sessionsMu     sync.RWMutex
	sessionsActive map[string]sessionContext
	location       locationstate.Location
}

// Event contains data about event, which is sent using transport
type Event struct {
	Application appInfo     `json:"application"`
	EventName   string      `json:"eventName"`
	CreatedAt   int64       `json:"createdAt"`
	Context     interface{} `json:"context"`
}

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

type natMappingContext struct {
	Stage        string              `json:"stage"`
	Successful   bool                `json:"successful"`
	ErrorMessage *string             `json:"error_message"`
	Gateways     []map[string]string `json:"gateways,omitempty"`
}

type sessionEventContext struct {
	Event string
	sessionContext
}

type sessionDataContext struct {
	Rx, Tx uint64
	sessionContext
}

type sessionTokensContext struct {
	Tokens *big.Int
	sessionContext
}

type registrationEvent struct {
	Identity string
	Status   string
	Country  string
}

type sessionTraceContext struct {
	Duration time.Duration
	Stage    string
	sessionContext
}

type sessionContext struct {
	ID              string
	Consumer        string
	Provider        string
	ServiceType     string
	ProviderCountry string
	ConsumerCountry string
	AccountantID    string
}

// Subscribe subscribes to relevant events of event bus.
func (sender *Sender) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionState, sender.sendConnStateEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionSession, sender.sendSessionEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(sessionEvent.AppTopicSession, sender.sendServiceSessionEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, sender.sendSessionData); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(pingpongEvent.AppTopicInvoicePaid, sender.sendSessionEarning); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(discovery.AppTopicProposalAnnounce, sender.sendProposalEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(trace.AppTopicTraceEvent, sender.sendTraceEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(registry.AppTopicIdentityRegistration, sender.sendRegistrationEvent); err != nil {
		return err
	}

	if err := bus.SubscribeAsync(location.LocUpdateEvent, sender.cacheLocationData); err != nil {
		return err
	}

	return bus.SubscribeAsync(identity.AppTopicIdentityUnlock, sender.sendUnlockEvent)
}

func (sender *Sender) cacheLocationData(l locationstate.Location) {
	sender.sessionsMu.RLock()
	defer sender.sessionsMu.RUnlock()

	sender.location = l
}

func (sender *Sender) getCachedLocationData() (l locationstate.Location) {
	sender.sessionsMu.RLock()
	defer sender.sessionsMu.RUnlock()
	l = sender.location
	return
}

// sendSessionData sends transferred information about session.
func (sender *Sender) sendSessionData(e connectionstate.AppEventConnectionStatistics) {
	if e.SessionInfo.SessionID == "" {
		return
	}

	sender.sendEvent(sessionDataName, sessionDataContext{
		Rx:             e.Stats.BytesReceived,
		Tx:             e.Stats.BytesSent,
		sessionContext: sender.toSessionContext(e.SessionInfo),
	})
}

func (sender *Sender) sendSessionEarning(e pingpongEvent.AppEventInvoicePaid) {
	session, err := sender.recoverSessionContext(e.SessionID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	sender.sendEvent(sessionTokensName, sessionTokensContext{
		Tokens:         e.Invoice.AgreementTotal,
		sessionContext: session,
	})
}

// sendConnStateEvent sends session update events.
func (sender *Sender) sendConnStateEvent(e connectionstate.AppEventConnectionState) {
	if e.SessionInfo.SessionID == "" {
		return
	}

	sender.sendEvent(sessionEventName, sessionEventContext{
		Event:          string(e.State),
		sessionContext: sender.toSessionContext(e.SessionInfo),
	})
}

func (sender *Sender) sendServiceSessionEvent(e sessionEvent.AppEventSession) {
	if e.Session.ID == "" {
		return
	}

	sessionContext := sessionContext{
		ID:              e.Session.ID,
		Consumer:        e.Session.ConsumerID.Address,
		Provider:        e.Session.Proposal.ProviderID,
		ServiceType:     e.Session.Proposal.ServiceType,
		ProviderCountry: e.Session.Proposal.ServiceDefinition.GetLocation().Country,
		ConsumerCountry: e.Session.ConsumerLocation.Country,
		AccountantID:    e.Session.HermesID.Hex(),
	}

	switch e.Status {
	case sessionEvent.CreatedStatus:
		sender.rememberSessionContext(sessionContext)
	case sessionEvent.RemovedStatus:
		sender.forgetSessionContext(sessionContext)
	}
}

// sendSessionEvent sends session update events.
func (sender *Sender) sendSessionEvent(e connectionstate.AppEventConnectionSession) {
	if e.SessionInfo.SessionID == "" {
		return
	}
	sessionContext := sender.toSessionContext(e.SessionInfo)

	switch e.Status {
	case connectionstate.SessionCreatedStatus:
		sender.rememberSessionContext(sessionContext)
		sender.sendEvent(sessionEventName, sessionEventContext{
			Event:          e.Status,
			sessionContext: sessionContext,
		})
	case connectionstate.SessionEndedStatus:
		sender.sendEvent(sessionEventName, sessionEventContext{
			Event:          e.Status,
			sessionContext: sessionContext,
		})
		sender.forgetSessionContext(sessionContext)
	}
}

// sendUnlockEvent sends startup event
func (sender *Sender) sendUnlockEvent(id string) {
	sender.sendEvent(unlockEventName, id)
}

// sendProposalEvent sends provider proposal event.
func (sender *Sender) sendProposalEvent(p market.ServiceProposal) {
	sender.sendEvent(proposalEventName, p)
}

func (sender *Sender) sendRegistrationEvent(r registry.AppEventIdentityRegistration) {
	l := sender.getCachedLocationData()
	sender.sendEvent(registerIdentity, registrationEvent{
		Identity: r.ID.Address,
		Status:   r.Status.String(),
		Country:  l.Country,
	})
}

func (sender *Sender) sendTraceEvent(stage trace.Event) {
	session, err := sender.recoverSessionContext(stage.ID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	sender.sendEvent(traceEventName, sessionTraceContext{
		Duration:       stage.Duration,
		Stage:          stage.Key,
		sessionContext: session,
	})
}

// SendNATMappingSuccessEvent sends event about successful NAT mapping
func (sender *Sender) SendNATMappingSuccessEvent(stage string, gateways []map[string]string) {
	sender.sendEvent(natMappingEventName, natMappingContext{
		Stage:      stage,
		Successful: true,
		Gateways:   gateways,
	})
}

// SendNATMappingFailEvent sends event about failed NAT mapping
func (sender *Sender) SendNATMappingFailEvent(stage string, gateways []map[string]string, err error) {
	errorMessage := err.Error()
	sender.sendEvent(natMappingEventName, natMappingContext{
		Stage:        stage,
		Successful:   false,
		ErrorMessage: &errorMessage,
		Gateways:     gateways,
	})
}

func (sender *Sender) sendEvent(eventName string, context interface{}) {
	err := sender.Transport.SendEvent(Event{
		Application: appInfo{
			Name:    appName,
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
			Version: sender.AppVersion,
		},
		EventName: eventName,
		CreatedAt: time.Now().Unix(),
		Context:   context,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send metric: " + eventName)
	}
}

func (sender *Sender) rememberSessionContext(context sessionContext) {
	sender.sessionsMu.Lock()
	defer sender.sessionsMu.Unlock()

	sender.sessionsActive[context.ID] = context
}

func (sender *Sender) forgetSessionContext(context sessionContext) {
	sender.sessionsMu.Lock()
	defer sender.sessionsMu.Unlock()

	delete(sender.sessionsActive, context.ID)
}

func (sender *Sender) recoverSessionContext(sessionID string) (sessionContext, error) {
	sender.sessionsMu.RLock()
	defer sender.sessionsMu.RUnlock()

	context, found := sender.sessionsActive[sessionID]
	if !found {
		return sessionContext{}, fmt.Errorf("unknown session: %s", sessionID)
	}

	return context, nil
}

func (sender *Sender) toSessionContext(session connectionstate.Status) sessionContext {
	return sessionContext{
		ID:              string(session.SessionID),
		Consumer:        session.ConsumerID.Address,
		Provider:        session.Proposal.ProviderID,
		ServiceType:     session.Proposal.ServiceType,
		ProviderCountry: session.Proposal.ServiceDefinition.GetLocation().Country,
		ConsumerCountry: session.ConsumerLocation.Country,
		AccountantID:    session.HermesID.Hex(),
	}
}
