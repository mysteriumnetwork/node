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
	"runtime"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/market"
	"github.com/rs/zerolog/log"
)

const (
	appName             = "myst"
	sessionDataName     = "session_data"
	sessionEventName    = "session_event"
	startupEventName    = "startup"
	proposalEventName   = "proposal_event"
	natMappingEventName = "nat_mapping"
)

// Transport allows sending events
type Transport interface {
	SendEvent(Event) error
}

// NewSender creates metrics sender with appropriate transport
func NewSender(transport Transport, appVersion string, manager connection.Manager, locationResolver location.OriginResolver) *Sender {
	return &Sender{
		Transport:  transport,
		AppVersion: appVersion,
		connection: manager,
		location:   locationResolver,
	}
}

// Sender builds events and sends them using given transport
type Sender struct {
	Transport      Transport
	AppVersion     string
	connection     connection.Manager
	location       location.OriginResolver
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

type sessionContext struct {
	ID              string
	Consumer        string
	Provider        string
	ServiceType     string
	ProviderCountry string
	ConsumerCountry string
}

// SendSessionData sends transferred information about session.
func (sender *Sender) SendSessionData(e connection.SessionStatsEvent) {
	currentSession := e.SessionInfo

	sender.sendEvent(sessionDataName, sessionDataContext{
		Rx: e.Stats.BytesReceived,
		Tx: e.Stats.BytesSent,
		sessionContext: sessionContext{
			ID:              string(currentSession.SessionID),
			Consumer:        currentSession.ConsumerID.Address,
			Provider:        currentSession.Proposal.ProviderID,
			ServiceType:     currentSession.Proposal.ServiceType,
			ProviderCountry: currentSession.Proposal.ServiceDefinition.GetLocation().Country,
			ConsumerCountry: sender.originCountry(),
		},
	})
}


// SendSessionEvent sends session update events.
func (sender *Sender) SendConnStateEvent(e connection.StateEvent) {
	sender.sendEvent(sessionEventName, sessionEventContext{
		Event: string(e.State),
		sessionContext: sessionContext{
			ID:              string(e.SessionInfo.SessionID),
			Consumer:        e.SessionInfo.ConsumerID.Address,
			Provider:        e.SessionInfo.Proposal.ProviderID,
			ServiceType:     e.SessionInfo.Proposal.ServiceType,
			ProviderCountry: e.SessionInfo.Proposal.ServiceDefinition.GetLocation().Country,
			ConsumerCountry: sender.originCountry(),
		},
	})
}

// SendSessionEvent sends session update events.
func (sender *Sender) SendSessionEvent(e connection.SessionEvent) {
	sender.sendEvent(sessionEventName, sessionEventContext{
		Event: e.Status,
		sessionContext: sessionContext{
			ID:              string(e.SessionInfo.SessionID),
			Consumer:        e.SessionInfo.ConsumerID.Address,
			Provider:        e.SessionInfo.Proposal.ProviderID,
			ServiceType:     e.SessionInfo.Proposal.ServiceType,
			ProviderCountry: e.SessionInfo.Proposal.ServiceDefinition.GetLocation().Country,
			ConsumerCountry: sender.originCountry(),
		},
	})
}

// SendStartupEvent sends startup event
func (sender *Sender) SendStartupEvent(e event.Payload) {
	sender.sendEvent(startupEventName, e.Status)
}

// SendProposalEvent sends provider proposal event.
func (sender *Sender) SendProposalEvent(p market.ServiceProposal) {
	sender.sendEvent(proposalEventName, p)
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

func (sender *Sender) originCountry() string {
	origin, err := sender.location.GetOrigin()
	if err != nil {
		log.Warn().Msg("Failed to get consumer origin country")
		return ""
	}
	return origin.Country
}
