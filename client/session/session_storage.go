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
	stats_dto "github.com/mysteriumnetwork/node/client/stats/dto"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

const sessionStorageLogPrefix = "[session-storage] "

// StatsRetriever can fetch current session stats
type StatsRetriever interface {
	Retrieve() stats_dto.SessionStats
}

// Storer allows us to get all sessions, save and update them
type Storer interface {
	Save(object interface{}) error
	Update(object interface{}) error
	GetAll(array interface{}) error
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

// Save saves a new session
func (repo *Storage) Save(se Session) error {
	return repo.storage.Save(&se)
}

// Update updates specified fields of existing session by id
func (repo *Storage) Update(sessionID node_session.ID, updated time.Time, dataStats stats_dto.SessionStats, status Status) error {
	// update fields by sessionID
	se := Session{SessionID: sessionID, Updated: updated, DataStats: dataStats, Status: status}
	return repo.storage.Update(&se)
}

// GetAll returns array of all sessions
func (repo *Storage) GetAll() ([]Session, error) {
	var sessions []Session
	err := repo.storage.GetAll(&sessions)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// ConsumeStateEvent consumes the connection state change events
func (repo *Storage) ConsumeStateEvent(stateEvent connection.StateEvent) {
	switch stateEvent.State {
	case connection.Disconnecting:
		err := repo.Update(stateEvent.SessionInfo.SessionID, time.Now(), repo.statsRetriever.Retrieve(), SessionStatusCompleted)
		if err != nil {
			log.Error(sessionStorageLogPrefix, err)
		} else {
			log.Trace(sessionStorageLogPrefix, fmt.Sprintf("Session %v updated", stateEvent.SessionInfo.SessionID))
		}
	case connection.Connected:
		providerCountry := stateEvent.SessionInfo.Proposal.ServiceDefinition.GetLocation().Country
		se := NewSession(
			stateEvent.SessionInfo.SessionID,
			identity.FromAddress(stateEvent.SessionInfo.Proposal.ProviderID),
			stateEvent.SessionInfo.Proposal.ServiceType,
			providerCountry,
		)
		err := repo.Save(*se)
		if err != nil {
			log.Error(sessionStorageLogPrefix, err)
		} else {
			log.Trace(sessionStorageLogPrefix, fmt.Sprintf("Session %v saved", stateEvent.SessionInfo.SessionID))
		}
	}
}
