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

// SessionsRepository describes functions for storing session objects
type SessionsRepository interface {
	Save(Session) error
	Update(session.ID, time.Duration, stats.SessionStats) error
	GetAll() ([]Session, error)
}

type sessionsRepository struct {
	storage storage.Storage
}

// NewSessionRepository creates session repository with given dependencies
func NewSessionRepository(storage storage.Storage) SessionsRepository {
	return &sessionsRepository{
		storage: storage,
	}
}

// Saves a new session
func (repo *sessionsRepository) Save(se Session) error {
	return repo.storage.Save(&se)
}

// Updates specified fields of existing session by id
func (repo *sessionsRepository) Update(sessionID session.ID, duration time.Duration, dataStats stats.SessionStats) error {
	// update two fields by sessionID
	se := Session{SessionID: sessionID, Duration: duration, DataStats: dataStats}
	return repo.storage.Update(&se)
}

func (repo *sessionsRepository) GetAll() ([]Session, error) {
	var sessions []Session
	err := repo.storage.GetAllSessions(&sessions)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
