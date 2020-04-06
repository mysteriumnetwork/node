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
	"github.com/mysteriumnetwork/node/session"
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

// Storage contains functions for storing, getting session objects
type Storage struct {
	storage      Storer
	statistics   connection.Statistics
	statisticsMu sync.RWMutex
}

// NewSessionStorage creates session repository with given dependencies
func NewSessionStorage(storage Storer) *Storage {
	return &Storage{
		storage: storage,
	}
}

// Subscribe subscribes to relevant events of event bus.
func (repo *Storage) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.Subscribe(connection.AppTopicConnectionSession, repo.consumeSessionEvent); err != nil {
		return err
	}
	return bus.Subscribe(connection.AppTopicConnectionStatistics, repo.consumeSessionStatisticsEvent)
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
		repo.statistics = connection.Statistics{}
		repo.handleCreatedEvent(sessionEvent.SessionInfo)
	}
}

func (repo *Storage) consumeSessionStatisticsEvent(e connection.AppEventConnectionStatistics) {
	repo.statisticsMu.Lock()
	repo.statistics = e.Stats
	repo.statisticsMu.Unlock()
}

func (repo *Storage) handleEndedEvent(sessionID session.ID) {
	repo.statisticsMu.RLock()
	dataStats := repo.statistics
	repo.statisticsMu.RUnlock()

	updatedSession := &History{
		SessionID: sessionID,
		Updated:   time.Now().UTC(),
		DataStats: dataStats,
		Status:    SessionStatusCompleted,
	}
	err := repo.storage.Update(sessionStorageBucketName, updatedSession)
	if err != nil {
		log.Error().Err(err).Msg("")
	} else {
		log.Debug().Msgf("Session %v updated", sessionID)
	}
}

func (repo *Storage) handleCreatedEvent(sessionInfo connection.Status) {
	se := NewHistory(
		sessionInfo.SessionID,
		sessionInfo.Proposal,
	)
	err := repo.storage.Store(sessionStorageBucketName, se)
	if err != nil {
		log.Error().Err(err).Msg("")
	} else {
		log.Debug().Msgf("Session %v saved", sessionInfo.SessionID)
	}
}
