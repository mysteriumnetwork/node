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
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/rs/zerolog/log"
)

const sessionStorageBucketName = "session-history"

// StatsRetriever can fetch current session stats
type StatsRetriever interface {
	GetDataStats() connection.Statistics
}

// Storer allows us to get all sessions, save and update them
type Storer interface {
	Store(bucket string, object interface{}) error
	Update(bucket string, object interface{}) error
	GetAllFrom(bucket string, array interface{}) error
}

type timeGetter func() time.Time

// Storage contains functions for storing, getting session objects
type Storage struct {
	storage    Storer
	timeGetter timeGetter

	mu             sync.RWMutex
	sessionsActive map[session.ID]History
}

// NewSessionStorage creates session repository with given dependencies
func NewSessionStorage(storage Storer) *Storage {
	return &Storage{
		storage:    storage,
		timeGetter: time.Now,

		sessionsActive: make(map[session.ID]History),
	}
}

// Subscribe subscribes to relevant events of pingpongEvent bus.
func (repo *Storage) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.Subscribe(connection.AppTopicConnectionSession, repo.consumeSessionEvent); err != nil {
		return err
	}
	if err := bus.Subscribe(connection.AppTopicConnectionStatistics, repo.consumeSessionStatisticsEvent); err != nil {
		return err
	}
	return bus.Subscribe(pingpongEvent.AppTopicInvoicePaid, repo.consumeSessionSpendingEvent)
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

// consumeSessionEvent consumes the session state change events
func (repo *Storage) consumeSessionEvent(sessionEvent connection.AppEventConnectionSession) {
	switch sessionEvent.Status {
	case connection.SessionEndedStatus:
		repo.handleEndedEvent(sessionEvent.SessionInfo.SessionID)
	case connection.SessionCreatedStatus:
		repo.handleCreatedEvent(sessionEvent.SessionInfo)
	}
}

func (repo *Storage) consumeSessionStatisticsEvent(e connection.AppEventConnectionStatistics) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	row, ok := repo.sessionsActive[e.SessionInfo.SessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}

	row.DataStats = e.Stats
	repo.sessionsActive[e.SessionInfo.SessionID] = row
}

func (repo *Storage) consumeSessionSpendingEvent(e pingpongEvent.AppEventInvoicePaid) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	sessionID := session.ID(e.SessionID)
	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msg("Received a unknown session update")
		return
	}
	row.Updated = repo.timeGetter().UTC()
	row.Invoice = e.Invoice

	err := repo.storage.Update(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v update failed", sessionID)
		return
	}

	repo.sessionsActive[sessionID] = row
	log.Debug().Msgf("Session %v updated", sessionID)
}

func (repo *Storage) handleEndedEvent(sessionID session.ID) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	row, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msgf("Can't find session %v to update", sessionID)
		return
	}
	row.Updated = repo.timeGetter().UTC()
	row.Status = SessionStatusCompleted

	err := repo.storage.Update(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v update failed", sessionID)
		return
	}

	delete(repo.sessionsActive, sessionID)
	log.Debug().Msgf("Session %v updated with final data", sessionID)
}

func (repo *Storage) handleCreatedEvent(session connection.Status) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	row := History{
		SessionID:       session.SessionID,
		ConsumerID:      session.ConsumerID,
		AccountantID:    session.AccountantID.Hex(),
		ProviderID:      identity.FromAddress(session.Proposal.ProviderID),
		ServiceType:     session.Proposal.ServiceType,
		ProviderCountry: session.Proposal.ServiceDefinition.GetLocation().Country,
		Started:         session.StartedAt.UTC(),
		Status:          SessionStatusNew,
	}
	err := repo.storage.Store(sessionStorageBucketName, &row)
	if err != nil {
		log.Error().Err(err).Msgf("Session %v insert failed", session.SessionID)
		return
	}

	repo.sessionsActive[session.SessionID] = row
	log.Debug().Msgf("Session %v saved", session.SessionID)
}
