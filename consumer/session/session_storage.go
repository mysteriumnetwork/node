/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	session_node "github.com/mysteriumnetwork/node/session"
	session_event "github.com/mysteriumnetwork/node/session/event"
	pingpong_event "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/rs/zerolog/log"
)

const sessionStorageBucketName = "session-history"

// StatsRetriever can fetch current session stats
type StatsRetriever interface {
	GetDataStats() connection.Statistics
}

type timeGetter func() time.Time

// Storage contains functions for storing, getting session objects
type Storage struct {
	storage    *boltdb.Bolt
	timeGetter timeGetter

	mu             sync.RWMutex
	sessionsActive map[session_node.ID]History
}

// NewSessionStorage creates session repository with given dependencies
func NewSessionStorage(storage *boltdb.Bolt) *Storage {
	return &Storage{
		storage:    storage,
		timeGetter: time.Now,

		sessionsActive: make(map[session_node.ID]History),
	}
}

// Subscribe subscribes to relevant events of event bus.
func (repo *Storage) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.Subscribe(session_event.AppTopicSession, repo.consumeServiceSessionEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(session_event.AppTopicDataTransferred, repo.consumeServiceSessionStatisticsEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(session_event.AppTopicTokensEarned, repo.consumeServiceSessionEarningsEvent); err != nil {
		return err
	}
	if err := bus.Subscribe(connection.AppTopicConnectionSession, repo.consumeConnectionSessionEvent); err != nil {
		return err
	}
	if err := bus.Subscribe(connection.AppTopicConnectionStatistics, repo.consumeConnectionStatisticsEvent); err != nil {
		return err
	}
	return bus.Subscribe(pingpong_event.AppTopicInvoicePaid, repo.consumeConnectionSpendingEvent)
}

// GetAll returns array of all sessions
func (repo *Storage) GetAll() ([]History, error) {
	var sessions []History
	err := repo.storage.GetAllFrom(sessionStorageBucketName, &sessions)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// consumeServiceSessionEvent consumes the provided sessions
func (repo *Storage) consumeServiceSessionEvent(e session_event.AppEventSession) {
	sessionID := session_node.ID(e.Session.ID)

	switch e.Status {
	case session_event.RemovedStatus:
		repo.handleEndedEvent(sessionID)
	case session_event.CreatedStatus:
		repo.mu.Lock()
		repo.sessionsActive[sessionID] = History{
			SessionID:       sessionID,
			Direction:       DirectionProvided,
			ConsumerID:      e.Session.ConsumerID,
			AccountantID:    e.Session.AccountantID.Hex(),
			ProviderID:      identity.FromAddress(e.Session.Proposal.ProviderID),
			ServiceType:     e.Session.Proposal.ServiceType,
			ProviderCountry: e.Session.Proposal.ServiceDefinition.GetLocation().Country,
			Started:         e.Session.StartedAt.UTC(),
		}
		repo.mu.Unlock()

		repo.handleCreatedEvent(sessionID)
	}
}

func (repo *Storage) consumeServiceSessionStatisticsEvent(e session_event.AppEventDataTransferred) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	sessionID := session_node.ID(e.ID)
	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}

	row.DataSent = e.Down
	row.DataReceived = e.Up
	repo.sessionsActive[sessionID] = row
}

func (repo *Storage) consumeServiceSessionEarningsEvent(e session_event.AppEventTokensEarned) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	sessionID := session_node.ID(e.SessionID)
	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}

	row.Tokens = e.Total
	repo.sessionsActive[sessionID] = row
}

// consumeConnectionSessionEvent consumes the session state change events
func (repo *Storage) consumeConnectionSessionEvent(e connection.AppEventConnectionSession) {
	sessionID := e.SessionInfo.SessionID

	switch e.Status {
	case connection.SessionEndedStatus:
		repo.handleEndedEvent(sessionID)
	case connection.SessionCreatedStatus:
		repo.mu.Lock()
		repo.sessionsActive[sessionID] = History{
			SessionID:       sessionID,
			Direction:       DirectionConsumed,
			ConsumerID:      e.SessionInfo.ConsumerID,
			AccountantID:    e.SessionInfo.AccountantID.Hex(),
			ProviderID:      identity.FromAddress(e.SessionInfo.Proposal.ProviderID),
			ServiceType:     e.SessionInfo.Proposal.ServiceType,
			ProviderCountry: e.SessionInfo.Proposal.ServiceDefinition.GetLocation().Country,
			Started:         e.SessionInfo.StartedAt.UTC(),
		}
		repo.mu.Unlock()

		repo.handleCreatedEvent(sessionID)
	}
}

func (repo *Storage) consumeConnectionStatisticsEvent(e connection.AppEventConnectionStatistics) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	row, ok := repo.sessionsActive[e.SessionInfo.SessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}

	row.DataSent = e.Stats.BytesSent
	row.DataReceived = e.Stats.BytesReceived
	repo.sessionsActive[e.SessionInfo.SessionID] = row
}

func (repo *Storage) consumeConnectionSpendingEvent(e pingpong_event.AppEventInvoicePaid) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	sessionID := session_node.ID(e.SessionID)
	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}
	row.Updated = repo.timeGetter().UTC()
	row.Tokens = e.Invoice.AgreementTotal

	err := repo.storage.Update(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v update failed", sessionID)
		return
	}

	repo.sessionsActive[sessionID] = row
	log.Debug().Msgf("Session %v updated", sessionID)
}

func (repo *Storage) handleEndedEvent(sessionID session_node.ID) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msgf("Can't find session %v to update", sessionID)
		return
	}
	row.Updated = repo.timeGetter().UTC()
	row.Status = StatusCompleted

	err := repo.storage.Update(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v update failed", sessionID)
		return
	}

	delete(repo.sessionsActive, sessionID)
	log.Debug().Msgf("Session %v updated with final data", sessionID)
}

func (repo *Storage) handleCreatedEvent(sessionID session_node.ID) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msgf("Can't find session %v to store", sessionID)
		return
	}
	row.Status = StatusNew

	err := repo.storage.Store(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v insert failed", row.SessionID)
		return
	}

	repo.sessionsActive[sessionID] = row
	log.Debug().Msgf("Session %v saved", row.SessionID)
}
