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

package connection

import (
	"time"

	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/storage"
	"github.com/mysteriumnetwork/node/session"
)

// SessionStorage contains functions for storing, getting session objects
type SessionStorage struct {
	storage storage.Storage
}

// NewSessionStorage creates session repository with given dependencies
func NewSessionStorage(storage storage.Storage) *SessionStorage {
	return &SessionStorage{
		storage: storage,
	}
}

// Save saves a new session
func (repo *SessionStorage) Save(se Session) error {
	return repo.storage.Save(&se)
}

// Update updates specified fields of existing session by id
func (repo *SessionStorage) Update(sessionID session.ID, updated time.Time, dataStats stats.SessionStats, status SessionStatus) error {
	// update fields by sessionID
	se := Session{SessionID: sessionID, Updated: updated, DataStats: dataStats, Status: status}
	return repo.storage.Update(&se)
}

// GetAll returns array of all sessions
func (repo *SessionStorage) GetAll() ([]Session, error) {
	var sessions []Session
	err := repo.storage.GetAll(&sessions)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
