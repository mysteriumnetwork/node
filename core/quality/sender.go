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
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/behavior"
	"github.com/mysteriumnetwork/node/p2p"
	p2pnat "github.com/mysteriumnetwork/node/p2p/nat"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/trace"
)

const (
	appName                  = "myst"
	connectionEvent          = "connection_event"
	sessionDataName          = "session_data"
	sessionTokensName        = "session_tokens"
	sessionEventName         = "session_event"
	traceEventName           = "trace_event"
	registerIdentity         = "register_identity"
	unlockEventName          = "unlock"
	proposalEventName        = "proposal_event"
	natMappingEventName      = "nat_mapping"
	pingEventName            = "ping_event"
	residentCountryEventName = "resident_country_event"
	stunDetectionEvent       = "stun_detection_event"
	natTypeDetectionEvent    = "nat_type_detection_event"
	natTraversalMethod       = "nat_traversal_method"
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

	identitiesMu       sync.RWMutex
	identitiesUnlocked []identity.Identity

	sessionsMu     sync.RWMutex
	sessionsActive map[string]sessionContext
}

// Event contains data about event, which is sent using transport
type Event struct {
	Application appInfo     `json:"application"`
	EventName   string      `json:"eventName"`
	CreatedAt   int64       `json:"createdAt"`
	Context     interface{} `json:"context"`
}

type appInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	LauncherVersion string `json:"launcher_version"`
	HostOS          string `json:"host_os"`
}

type pingEventContext struct {
	IsProvider bool
	Duration   uint64
	sessionContext
}

type natMappingContext struct {
	ID           string              `json:"id"`
	Stage        string              `json:"stage"`
	Successful   bool                `json:"successful"`
	ErrorMessage *string             `json:"error_message"`
	Gateways     []map[string]string `json:"gateways,omitempty"`
}

type sessionEventContext struct {
	IsProvider bool
	Event      string
	sessionContext
}

type sessionDataContext struct {
	IsProvider bool
	Rx, Tx     uint64
	sessionContext
}

type sessionTokensContext struct {
	Tokens *big.Int
	sessionContext
}

type registrationEvent struct {
	Identity string
	Status   string
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
	StartedAt       time.Time
}

type residentCountryEvent struct {
	ID      string
	Country string
}

type natTypeEvent struct {
	ID      string
	NATType string
}

type natMethodEvent struct {
	ID        string
	NATMethod string
	Success   bool
}

// Subscribe subscribes to relevant events of event bus.
func (s *Sender) Subscribe(bus eventbus.Subscriber) error {
	subscription := map[string]interface{}{
		AppTopicConnectionEvents:                     s.sendConnectionEvent,
		connectionstate.AppTopicConnectionState:      s.sendConnStateEvent,
		connectionstate.AppTopicConnectionSession:    s.sendSessionEvent,
		connectionstate.AppTopicConnectionStatistics: s.sendSessionData,
		discovery.AppTopicProposalAnnounce:           s.sendProposalEvent,
		identity.AppTopicIdentityUnlock:              s.sendUnlockEvent,
		pingpongEvent.AppTopicInvoicePaid:            s.sendSessionEarning,
		registry.AppTopicIdentityRegistration:        s.sendRegistrationEvent,
		sessionEvent.AppTopicSession:                 s.sendServiceSessionEvent,
		trace.AppTopicTraceEvent:                     s.sendTraceEvent,
		sessionEvent.AppTopicDataTransferred:         s.sendServiceDataStatistics,
		AppTopicConsumerPingP2P:                      s.sendConsumerPingDistance,
		AppTopicProviderPingP2P:                      s.sendProviderPingDistance,
		identity.AppTopicResidentCountry:             s.sendResidentCountry,
		p2p.AppTopicSTUN:                             s.sendSTUNDetectionStatus,
		behavior.AppTopicNATTypeDetected:             s.sendNATType,
		p2pnat.AppTopicNATTraversalMethod:            s.sendNATtraversalMethod,
	}

	for topic, fn := range subscription {
		if err := bus.SubscribeAsync(topic, fn); err != nil {
			return err
		}
	}

	return nil
}

func (s *Sender) sendNATtraversalMethod(method p2pnat.NATTraversalMethod) {
	s.sendEvent(natTraversalMethod, natMethodEvent{
		ID:        method.Identity,
		NATMethod: method.Method,
		Success:   method.Success,
	})
}

func (s *Sender) sendNATType(natType nat.NATType) {
	s.identitiesMu.RLock()
	defer s.identitiesMu.RUnlock()

	for _, id := range s.identitiesUnlocked {
		s.sendEvent(natTypeDetectionEvent, natTypeEvent{
			ID:      id.Address,
			NATType: string(natType),
		})
	}
}

func (s *Sender) sendSTUNDetectionStatus(status p2p.STUNDetectionStatus) {
	s.sendEvent(stunDetectionEvent, natTypeEvent{
		ID:      status.Identity,
		NATType: status.NATType,
	})
}

func (s *Sender) sendResidentCountry(e identity.ResidentCountryEvent) {
	s.sendEvent(residentCountryEventName, residentCountryEvent{
		ID:      e.ID,
		Country: e.Country,
	})
}

func (s *Sender) sendConsumerPingDistance(p PingEvent) {
	s.sendPingDistance(false, p)
}

func (s *Sender) sendProviderPingDistance(p PingEvent) {
	s.sendPingDistance(true, p)
}

func (s *Sender) sendPingDistance(isProvider bool, p PingEvent) {
	session, err := s.recoverSessionContext(p.SessionID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	s.sendEvent(pingEventName, pingEventContext{
		IsProvider:     isProvider,
		Duration:       uint64(p.Duration),
		sessionContext: session,
	})
}

func (s *Sender) sendConnectionEvent(e ConnectionEvent) {
	s.sendEvent(connectionEvent, e)
}

func (s *Sender) sendServiceDataStatistics(e sessionEvent.AppEventDataTransferred) {
	session, err := s.recoverSessionContext(e.ID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	s.sendEvent(sessionDataName, sessionDataContext{
		IsProvider:     true,
		Rx:             e.Up,
		Tx:             e.Down,
		sessionContext: session,
	})
}

// sendSessionData sends transferred information about session.
func (s *Sender) sendSessionData(e connectionstate.AppEventConnectionStatistics) {
	if e.SessionInfo.SessionID == "" {
		return
	}

	s.sendEvent(sessionDataName, sessionDataContext{
		IsProvider:     false,
		Rx:             e.Stats.BytesReceived,
		Tx:             e.Stats.BytesSent,
		sessionContext: s.toSessionContext(e.SessionInfo),
	})
}

func (s *Sender) sendSessionEarning(e pingpongEvent.AppEventInvoicePaid) {
	session, err := s.recoverSessionContext(e.SessionID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	s.sendEvent(sessionTokensName, sessionTokensContext{
		Tokens:         e.Invoice.AgreementTotal,
		sessionContext: session,
	})
}

// sendConnStateEvent sends session update events.
func (s *Sender) sendConnStateEvent(e connectionstate.AppEventConnectionState) {
	if e.SessionInfo.SessionID == "" {
		return
	}

	s.sendEvent(sessionEventName, sessionEventContext{
		Event:          string(e.State),
		sessionContext: s.toSessionContext(e.SessionInfo),
	})
}

func (s *Sender) sendServiceSessionEvent(e sessionEvent.AppEventSession) {
	if e.Session.ID == "" {
		return
	}

	sessionContext := sessionContext{
		ID:              e.Session.ID,
		Consumer:        e.Session.ConsumerID.Address,
		Provider:        e.Session.Proposal.ProviderID,
		ServiceType:     e.Session.Proposal.ServiceType,
		ProviderCountry: e.Session.Proposal.Location.Country,
		ConsumerCountry: e.Session.ConsumerLocation.Country,
		AccountantID:    e.Session.HermesID.Hex(),
		StartedAt:       e.Session.StartedAt,
	}

	switch e.Status {
	case sessionEvent.CreatedStatus:
		s.rememberSessionContext(sessionContext)
	case sessionEvent.RemovedStatus:
		s.forgetSessionContext(sessionContext)
	}

	s.sendEvent(sessionEventName, sessionEventContext{
		IsProvider:     true,
		Event:          string(e.Status),
		sessionContext: sessionContext,
	})
}

// sendSessionEvent sends session update events.
func (s *Sender) sendSessionEvent(e connectionstate.AppEventConnectionSession) {
	if e.SessionInfo.SessionID == "" {
		return
	}

	sessionContext := s.toSessionContext(e.SessionInfo)

	switch e.Status {
	case connectionstate.SessionCreatedStatus:
		s.rememberSessionContext(sessionContext)
		s.sendEvent(sessionEventName, sessionEventContext{
			IsProvider:     false,
			Event:          e.Status,
			sessionContext: sessionContext,
		})
	case connectionstate.SessionEndedStatus:
		s.sendEvent(sessionEventName, sessionEventContext{
			IsProvider:     false,
			Event:          e.Status,
			sessionContext: sessionContext,
		})
		s.forgetSessionContext(sessionContext)
	}
}

// sendUnlockEvent sends startup event
func (s *Sender) sendUnlockEvent(ev identity.AppEventIdentityUnlock) {
	s.identitiesMu.Lock()
	defer s.identitiesMu.Unlock()
	s.identitiesUnlocked = append(s.identitiesUnlocked, ev.ID)

	s.sendEvent(unlockEventName, ev.ID.Address)
}

// sendProposalEvent sends provider proposal event.
func (s *Sender) sendProposalEvent(p market.ServiceProposal) {
	s.sendEvent(proposalEventName, p)
}

func (s *Sender) sendRegistrationEvent(r registry.AppEventIdentityRegistration) {
	s.sendEvent(registerIdentity, registrationEvent{
		Identity: r.ID.Address,
		Status:   r.Status.String(),
	})
}

func (s *Sender) sendTraceEvent(stage trace.Event) {
	session, err := s.recoverSessionContext(stage.ID)
	if err != nil {
		log.Warn().Err(err).Msg("Can't recover session context")
		return
	}

	s.sendEvent(traceEventName, sessionTraceContext{
		Duration:       stage.Duration,
		Stage:          stage.Key,
		sessionContext: session,
	})
}

// SendNATMappingSuccessEvent sends event about successful NAT mapping
func (s *Sender) SendNATMappingSuccessEvent(id, stage string, gateways []map[string]string) {
	s.sendEvent(natMappingEventName, natMappingContext{
		ID:         id,
		Stage:      stage,
		Successful: true,
		Gateways:   gateways,
	})
}

// SendNATMappingFailEvent sends event about failed NAT mapping
func (s *Sender) SendNATMappingFailEvent(id, stage string, gateways []map[string]string, err error) {
	errorMessage := err.Error()

	s.sendEvent(natMappingEventName, natMappingContext{
		ID:           id,
		Stage:        stage,
		Successful:   false,
		ErrorMessage: &errorMessage,
		Gateways:     gateways,
	})
}

func (s *Sender) sendEvent(eventName string, context interface{}) {
	guestOS := runtime.GOOS
	if _, err := os.Stat("/.dockerenv"); err == nil {
		guestOS += "(docker)"
	}
	launcherInfo := strings.Split(config.GetString(config.FlagLauncherVersion), "/")
	launcherVersion := launcherInfo[0]
	hostOS := ""
	if len(launcherInfo) > 1 {
		hostOS = launcherInfo[1]
	}

	err := s.Transport.SendEvent(Event{
		Application: appInfo{
			Name:            appName,
			OS:              guestOS,
			Arch:            runtime.GOARCH,
			Version:         s.AppVersion,
			LauncherVersion: launcherVersion,
			HostOS:          hostOS,
		},
		EventName: eventName,
		CreatedAt: time.Now().Unix(),
		Context:   context,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send metric: " + eventName)
	}
}

func (s *Sender) rememberSessionContext(context sessionContext) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	s.sessionsActive[context.ID] = context
}

func (s *Sender) forgetSessionContext(context sessionContext) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	delete(s.sessionsActive, context.ID)
}

func (s *Sender) recoverSessionContext(sessionID string) (sessionContext, error) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	context, found := s.sessionsActive[sessionID]
	if !found {
		return sessionContext{}, fmt.Errorf("unknown session: %s", sessionID)
	}

	return context, nil
}

func (s *Sender) toSessionContext(session connectionstate.Status) sessionContext {
	return sessionContext{
		ID:              string(session.SessionID),
		Consumer:        session.ConsumerID.Address,
		Provider:        session.Proposal.ProviderID,
		ServiceType:     session.Proposal.ServiceType,
		ProviderCountry: session.Proposal.Location.Country,
		ConsumerCountry: session.ConsumerLocation.Country,
		AccountantID:    session.HermesID.Hex(),
		StartedAt:       session.StartedAt,
	}
}
