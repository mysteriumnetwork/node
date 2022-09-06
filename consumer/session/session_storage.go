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
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	session_node "github.com/mysteriumnetwork/node/session"
	session_event "github.com/mysteriumnetwork/node/session/event"
	pingpong_event "github.com/mysteriumnetwork/node/session/pingpong/event"
)

const sessionStorageBucketName = "session-history"

type timeGetter func() time.Time

// Storage contains functions for storing, getting session objects.
type Storage struct {
	storage    *boltdb.Bolt
	timeGetter timeGetter

	mu             sync.RWMutex
	sessionsActive map[session_node.ID]History
}

// NewSessionStorage creates session repository with given dependencies.
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
	if err := bus.Subscribe(connectionstate.AppTopicConnectionSession, repo.consumeConnectionSessionEvent); err != nil {
		return err
	}
	if err := bus.Subscribe(connectionstate.AppTopicConnectionStatistics, repo.consumeConnectionStatisticsEvent); err != nil {
		return err
	}
	return bus.Subscribe(pingpong_event.AppTopicInvoicePaid, repo.consumeConnectionSpendingEvent)
}

// GetAll returns array of all sessions.
func (repo *Storage) GetAll() ([]History, error) {
	return repo.List(NewFilter())
}

// List retrieves stored entries.
func (repo *Storage) List(filter *Filter) (result []History, err error) {
	repo.storage.RLock()
	defer repo.storage.RUnlock()
	query := repo.storage.DB().
		From(sessionStorageBucketName).
		Select(filter.toMatcher()).
		OrderBy("Started").
		Reverse()

	err = query.Find(&result)
	if errors.Is(err, storm.ErrNotFound) {
		return []History{}, nil
	}

	return result, err
}

// Stats fetches aggregated statistics to Filter.Stats.
func (repo *Storage) Stats(filter *Filter) (result Stats, err error) {
	repo.storage.RLock()
	defer repo.storage.RUnlock()
	query := repo.storage.DB().
		From(sessionStorageBucketName).
		Select(filter.toMatcher()).
		OrderBy("Started").
		Reverse()

	result = NewStats()
	err = query.Each(new(History), func(record interface{}) error {
		session := record.(*History)

		result.Add(*session)

		return nil
	})
	return result, err
}

const stepDay = 24 * time.Hour

// StatsByDay retrieves aggregated statistics grouped by day to Filter.StatsByDay.
func (repo *Storage) StatsByDay(filter *Filter) (result map[time.Time]Stats, err error) {
	repo.storage.RLock()
	defer repo.storage.RUnlock()
	query := repo.storage.DB().
		From(sessionStorageBucketName).
		Select(filter.toMatcher()).
		OrderBy("Started").
		Reverse()

	// fill the period with zeros
	result = make(map[time.Time]Stats)
	if filter.StartedFrom != nil && filter.StartedTo != nil {
		for i := filter.StartedFrom.Truncate(stepDay); !i.After(*filter.StartedTo); i = i.Add(stepDay) {
			result[i] = NewStats()
		}
	}

	err = query.Each(new(History), func(record interface{}) error {
		session := record.(*History)

		i := session.Started.Truncate(stepDay)
		stats := result[i]
		stats.Add(*session)
		result[i] = stats

		return nil
	})
	return result, err
}

// consumeServiceSessionEvent consumes the provided sessions.
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
			HermesID:        e.Session.HermesID.Hex(),
			ProviderID:      identity.FromAddress(e.Session.Proposal.ProviderID),
			ServiceType:     e.Session.Proposal.ServiceType,
			ConsumerCountry: e.Session.ConsumerLocation.Country,
			ProviderCountry: e.Session.Proposal.Location.Country,
			Started:         e.Session.StartedAt.UTC(),
			Tokens:          new(big.Int),
		}
		repo.mu.Unlock()

		repo.handleCreatedEvent(sessionID)
	}
}

func (repo *Storage) consumeServiceSessionStatisticsEvent(e session_event.AppEventDataTransferred) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	sessionID := session_node.ID(e.ID)
	row, ok := repo.activeSession(sessionID)
	if !ok {
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
	row, ok := repo.activeSession(sessionID)
	if !ok {
		return
	}

	if big.NewInt(0).Cmp(e.Total) == 0 {
		log.Debug().Fields(map[string]interface{}{
			"sessionID":    sessionID,
			"consumerID":   row.ConsumerID,
			"providerID":   row.ProviderID,
			"dataReceived": row.DataReceived,
			"dataSent":     row.DataSent,
			"duration":     row.GetDuration(),
		}).Msgf("Zero earning event")
	}
	row.Tokens = e.Total
	repo.sessionsActive[sessionID] = row
}

func (repo *Storage) activeSession(sessionID session_node.ID) (History, bool) {
	history, ok := repo.sessionsActive[sessionID]
	if !ok {
		log.Warn().Msgf("Received a unknown session update, sessionID: %s", sessionID)
		return History{}, false
	}
	return history, true
}

// consumeConnectionSessionEvent consumes the session state change events
func (repo *Storage) consumeConnectionSessionEvent(e connectionstate.AppEventConnectionSession) {
	sessionID := e.SessionInfo.SessionID

	switch e.Status {
	case connectionstate.SessionEndedStatus:
		repo.handleEndedEvent(sessionID)
	case connectionstate.SessionCreatedStatus:
		repo.mu.Lock()
		repo.sessionsActive[sessionID] = History{
			SessionID:       sessionID,
			Direction:       DirectionConsumed,
			ConsumerID:      e.SessionInfo.ConsumerID,
			HermesID:        e.SessionInfo.HermesID.Hex(),
			ProviderID:      identity.FromAddress(e.SessionInfo.Proposal.ProviderID),
			ServiceType:     e.SessionInfo.Proposal.ServiceType,
			ConsumerCountry: e.SessionInfo.ConsumerLocation.Country,
			ProviderCountry: e.SessionInfo.Proposal.Location.Country,
			Started:         e.SessionInfo.StartedAt.UTC(),
			IPType:          e.SessionInfo.Proposal.Location.IPType,
			Tokens:          new(big.Int),
		}
		repo.mu.Unlock()

		repo.handleCreatedEvent(sessionID)
	}
}

func (repo *Storage) consumeConnectionStatisticsEvent(e connectionstate.AppEventConnectionStatistics) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	row, ok := repo.activeSession(e.SessionInfo.SessionID)
	if !ok {
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
	row, ok := repo.activeSession(sessionID)
	if !ok {
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
	repo.mu.Lock()
	defer repo.mu.Unlock()

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
