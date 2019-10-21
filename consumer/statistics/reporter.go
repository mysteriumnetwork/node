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
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
)

const statsSenderLogPrefix = "[session-stats-sender] "

// ErrSessionNotStarted represents the error that occurs when the session has not been started yet
var ErrSessionNotStarted = errors.New("session not started")

// StatsTracker allows for retrieval and resetting of statistics
type StatsTracker interface {
	Retrieve() consumer.SessionStatistics
	Reset()
}

// Reporter defines method for sending stats outside
// TODO probably bad naming needs improvement or better definition of our statistics server
type Reporter interface {
	SendSessionStats(session.ID, mysterium.SessionStats, identity.Signer) error
}

// SessionStatisticsReporter sends session stats to remote API server with a fixed sendInterval.
// Extra one send will be done on session disconnect.
type SessionStatisticsReporter struct {
	locationDetector location.OriginResolver

	signerFactory     identity.SignerFactory
	statisticsTracker StatsTracker
	remoteReporter    Reporter

	sendInterval time.Duration
	done         chan struct{}

	opLock  sync.Mutex
	started bool
}

// NewSessionStatisticsReporter function creates new session stats sender by given options
func NewSessionStatisticsReporter(statisticsTracker StatsTracker, remoteReporter Reporter, signerFactory identity.SignerFactory, locationDetector location.OriginResolver, interval time.Duration) *SessionStatisticsReporter {
	return &SessionStatisticsReporter{
		locationDetector:  locationDetector,
		signerFactory:     signerFactory,
		statisticsTracker: statisticsTracker,
		remoteReporter:    remoteReporter,

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
	loc, err := sr.locationDetector.GetOrigin()
	if err != nil {
		log.Error(statsSenderLogPrefix, "Failed to resolve location: ", err)
	}

	sr.done = make(chan struct{})

	go func() {
		for {
			select {
			case <-sr.done:
				if err := sr.send(serviceType, providerID, loc.Country, sessionID, signer); err != nil {
					log.Error(statsSenderLogPrefix, "Failed to send session stats to the remote service: ", err)
				} else {
					log.Debug(statsSenderLogPrefix, "Final stats sent")
				}
				// reset the stats in preparation for a new session
				sr.statisticsTracker.Reset()
				return
			case <-time.After(sr.sendInterval):
				if err := sr.send(serviceType, providerID, loc.Country, sessionID, signer); err != nil {
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
	sessionStats := sr.statisticsTracker.Retrieve()
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

// ConsumeSessionEvent handles the session state changes
func (sr *SessionStatisticsReporter) ConsumeSessionEvent(sessionEvent connection.SessionEvent) {
	switch sessionEvent.Status {
	case connection.SessionEventStatusEnded:
		sr.stop()
	case connection.SessionEventStatusCreated:
		sr.start(
			sessionEvent.SessionInfo.ConsumerID,
			sessionEvent.SessionInfo.Proposal.ServiceType,
			sessionEvent.SessionInfo.Proposal.ProviderID,
			sessionEvent.SessionInfo.SessionID,
		)
	}
}
