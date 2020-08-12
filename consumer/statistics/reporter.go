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

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/session"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ErrSessionNotStarted represents the error that occurs when the session has not been started yet
var ErrSessionNotStarted = errors.New("session not started")

// StatsTracker allows for retrieval and resetting of statistics
type StatsTracker interface {
	GetDataStats() connectionstate.Statistics
}

// Reporter defines method for sending stats outside
// TODO probably bad naming needs improvement or better definition of our statistics server
type Reporter interface {
	SendSessionStats(session.ID, mysterium.SessionStats, identity.Signer) error
}

// SessionStatisticsReporter sends session stats to remote API server with a fixed sendInterval.
// Extra one send will be done on session disconnect.
type SessionStatisticsReporter struct {
	signerFactory  identity.SignerFactory
	statistics     connectionstate.Statistics
	statisticsMu   sync.RWMutex
	remoteReporter Reporter

	sendInterval time.Duration
	done         chan struct{}

	opLock  sync.Mutex
	started bool
}

// NewSessionStatisticsReporter function creates new session stats sender by given options
func NewSessionStatisticsReporter(remoteReporter Reporter, signerFactory identity.SignerFactory, interval time.Duration) *SessionStatisticsReporter {
	return &SessionStatisticsReporter{
		signerFactory:  signerFactory,
		remoteReporter: remoteReporter,

		sendInterval: interval,
		done:         make(chan struct{}),
	}
}

// Subscribe subscribes to relevant events of event bus.
func (sr *SessionStatisticsReporter) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.Subscribe(connectionstate.AppTopicConnectionSession, sr.consumeSessionEvent); err != nil {
		return err
	}
	return bus.Subscribe(connectionstate.AppTopicConnectionStatistics, sr.consumeSessionStatisticsEvent)
}

// start starts sending of stats
func (sr *SessionStatisticsReporter) start(session connectionstate.Status) {
	sr.opLock.Lock()
	defer sr.opLock.Unlock()

	if sr.started {
		return
	}

	signer := sr.signerFactory(session.ConsumerID)

	sr.done = make(chan struct{})
	go func() {
		for {
			select {
			case <-sr.done:
				if err := sr.send(session, signer); err != nil {
					log.Error().Err(err).Msg("Failed to send session stats to the remote service")
				} else {
					log.Debug().Msg("Final stats sent")
				}
				return
			case <-time.After(sr.sendInterval):
				if err := sr.send(session, signer); err != nil {
					log.Error().Err(err).Msg("Failed to send session stats to the remote service")
				} else {
					log.Debug().Msg("Stats sent")
				}
			}
		}
	}()

	sr.started = true
	log.Debug().Msg("Session statistics reporter started")
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
	log.Debug().Msg("Session statistics reporter stopping")
}

func (sr *SessionStatisticsReporter) send(session connectionstate.Status, signer identity.Signer) error {
	sr.statisticsMu.RLock()
	dataStats := sr.statistics
	sr.statisticsMu.RUnlock()

	return sr.remoteReporter.SendSessionStats(
		session.SessionID,
		mysterium.SessionStats{
			ServiceType:     session.Proposal.ServiceType,
			BytesSent:       dataStats.BytesSent,
			BytesReceived:   dataStats.BytesReceived,
			ProviderID:      session.Proposal.ProviderID,
			ConsumerCountry: session.ConsumerLocation.Country,
		},
		signer,
	)
}

// consumeSessionEvent handles the session state changes
func (sr *SessionStatisticsReporter) consumeSessionEvent(sessionEvent connectionstate.AppEventConnectionSession) {
	switch sessionEvent.Status {
	case connectionstate.SessionEndedStatus:
		sr.stop()
	case connectionstate.SessionCreatedStatus:
		sr.statistics = connectionstate.Statistics{}
		sr.start(sessionEvent.SessionInfo)
	}
}

func (sr *SessionStatisticsReporter) consumeSessionStatisticsEvent(e connectionstate.AppEventConnectionStatistics) {
	sr.statisticsMu.Lock()
	sr.statistics = e.Stats
	sr.statisticsMu.Unlock()
}
