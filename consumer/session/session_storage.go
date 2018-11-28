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
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

const sessionStorageLogPrefix = "[session-storage] "
const sessionStorageBucketName = "session-history"

// StatsRetriever can fetch current session stats
type StatsRetriever interface {
	Retrieve() consumer.SessionStatistics
}

// Storer allows us to get all sessions, save and update them
type Storer interface {
	Store(bucket string, object interface{}) error
	Update(bucket string, object interface{}) error
	GetAllFrom(bucket string, array interface{}) error
}

// Storage contains functions for storing, getting session objects
type Storage struct {
	storage        Storer
	statsRetriever StatsRetriever
}

// NewSessionStorage creates session repository with given dependencies
func NewSessionStorage(storage Storer, statsRetriever StatsRetriever) *Storage {
	return &Storage{
		storage:        storage,
		statsRetriever: statsRetriever,
	}
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

// ConsumeStateEvent consumes the connection state change events
func (repo *Storage) ConsumeStateEvent(stateEvent connection.StateEvent) {
	switch stateEvent.State {
	case connection.Disconnecting:
		repo.handleDisconnectingState(stateEvent.SessionInfo.SessionID)
	case connection.Connected:
		repo.handleConnectedEvent(stateEvent.SessionInfo)
	}
}

func (repo *Storage) handleDisconnectingState(sessionID session.ID) {
	updatedSession := &History{
		SessionID: sessionID,
		Updated:   time.Now().UTC(),
		DataStats: repo.statsRetriever.Retrieve(),
		Status:    SessionStatusCompleted,
	}
	err := repo.storage.Update(sessionStorageBucketName, updatedSession)
	if err != nil {
		log.Error(sessionStorageLogPrefix, err)
	} else {
		log.Trace(sessionStorageLogPrefix, fmt.Sprintf("Session %v updated", sessionID))
	}
}

func (repo *Storage) handleConnectedEvent(sessionInfo connection.SessionInfo) {
	providerCountry := sessionInfo.Proposal.ServiceDefinition.GetLocation().Country
	se := NewHistory(
		sessionInfo.SessionID,
		identity.FromAddress(sessionInfo.Proposal.ProviderID),
		sessionInfo.Proposal.ServiceType,
		providerCountry,
	)
	err := repo.storage.Store(sessionStorageBucketName, se)
	if err != nil {
		log.Error(sessionStorageLogPrefix, err)
	} else {
		log.Trace(sessionStorageLogPrefix, fmt.Sprintf("Session %v saved", sessionInfo.SessionID))
	}
}
