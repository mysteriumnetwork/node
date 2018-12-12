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

package statistics

import (
	"errors"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/session"
)

const statsSenderLogPrefix = "[session-stats-sender] "

// ErrSessionNotStarted represents the error that occurs when the session has not been started yet
var ErrSessionNotStarted = errors.New("session not started")

// LocationDetector detects the country for session stats
type LocationDetector func() location.Location

// Retriever allows for retrieval of statistics
type Retriever interface {
	Retrieve() consumer.SessionStatistics
}

// Reporter defines method for sending stats outside
// TODO probably bad naming needs improvement or better definition of our statistics server
type Reporter interface {
	SendSessionStats(session.ID, mysterium.SessionStats, identity.Signer) error
}

// SessionStatisticsReporter sends session stats to remote API server with a fixed sendInterval.
// Extra one send will be done on session disconnect.
type SessionStatisticsReporter struct {
	locationDetector LocationDetector

	signerFactory       identity.SignerFactory
	statisticsRetriever Retriever
	remoteReporter      Reporter

	sendInterval time.Duration
	done         chan struct{}

	opLock  sync.Mutex
	started bool
}

// NewSessionStatisticsReporter function creates new session stats sender by given options
func NewSessionStatisticsReporter(statisticsRetriever Retriever, remoteReporter Reporter, signerFactory identity.SignerFactory, locationDetector LocationDetector, interval time.Duration) *SessionStatisticsReporter {
	return &SessionStatisticsReporter{
		locationDetector:    locationDetector,
		signerFactory:       signerFactory,
		statisticsRetriever: statisticsRetriever,
		remoteReporter:      remoteReporter,

		sendInterval: interval,
		done:         make(chan struct{}),
	}
}

// start starts sending of stats
func (sr *SessionStatisticsReporter) start(consumerID identity.Identity, serviceType, providerID string, sessionID session.ID) {
	sr.opLock.Lock()
	defer sr.opLock.Unlock()

	if sr.started {
		return
	}

	signer := sr.signerFactory(consumerID)
	country := sr.locationDetector().Country
	sr.done = make(chan struct{})

	go func() {
		for {
			select {
			case <-sr.done:
				if err := sr.send(serviceType, providerID, country, sessionID, signer); err != nil {
					log.Error(statsSenderLogPrefix, "Failed to send session stats to the remote service: ", err)
				} else {
					log.Debug(statsSenderLogPrefix, "Final stats sent")
				}
				return
			case <-time.After(sr.sendInterval):
				if err := sr.send(serviceType, providerID, country, sessionID, signer); err != nil {
					log.Error(statsSenderLogPrefix, "Failed to send session stats to the remote service: ", err)
				} else {
					log.Debug(statsSenderLogPrefix, "Stats sent")
				}
			}
		}
	}()

	sr.started = true
	log.Debug(statsSenderLogPrefix, "started")
}

// stop stops the sending of stats
func (sr *SessionStatisticsReporter) stop() {
	sr.opLock.Lock()
	defer sr.opLock.Unlock()

	if !sr.started {
		return
	}

	close(sr.done)
	sr.started = false
	log.Debug(statsSenderLogPrefix, "stopping")
}

func (sr *SessionStatisticsReporter) send(serviceType, providerID, country string, sessionID session.ID, signer identity.Signer) error {
	sessionStats := sr.statisticsRetriever.Retrieve()
	return sr.remoteReporter.SendSessionStats(
		sessionID,
		mysterium.SessionStats{
			ServiceType:     serviceType,
			BytesSent:       sessionStats.BytesSent,
			BytesReceived:   sessionStats.BytesReceived,
			ProviderID:      providerID,
			ConsumerCountry: country,
		},
		signer,
	)
}

// ConsumeStateEvent handles the connection state changes
func (sr *SessionStatisticsReporter) ConsumeStateEvent(stateEvent connection.StateEvent) {
	switch stateEvent.State {
	case connection.Disconnecting:
		sr.stop()
	case connection.Connected:
		sr.start(
			stateEvent.SessionInfo.ConsumerID,
			stateEvent.SessionInfo.Proposal.ServiceType,
			stateEvent.SessionInfo.Proposal.ProviderID,
			stateEvent.SessionInfo.SessionID,
		)
	}
}
