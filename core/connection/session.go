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
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

// SessionStatus represents list of possible session statuses
type SessionStatus int

const (
	// SessionStatusNew means that newly created session object is written to storage
	SessionStatusNew = SessionStatus(0)
	// SessionStatusCompleted means that session object is updated on connection disconnect event
	SessionStatusCompleted = SessionStatus(1)
)

// String converts status constant to string
func (st *SessionStatus) String() string {
	switch *st {
	case SessionStatusNew:
		return "New"
	case SessionStatusCompleted:
		return "Completed"
	}
	return ""
}

// NewSession creates session with given dependencies
func NewSession(sessionID session.ID, providerID identity.Identity, serviceType string, providerCountry string) *Session {
	return &Session{
		SessionID:       sessionID,
		ProviderID:      providerID,
		ServiceType:     serviceType,
		ProviderCountry: providerCountry,
		Started:         time.Now(),
		Status:          SessionStatusNew,
	}
}

// Session holds structure for saving session history
type Session struct {
	SessionID       session.ID `storm:"id"`
	ProviderID      identity.Identity
	ServiceType     string
	ProviderCountry string
	Started         time.Time
	Status          SessionStatus
	Updated         time.Time
	DataStats       stats.SessionStats // is updated on disconnect event
}

// GetDuration returns delta in seconds (TimeUpdated - TimeStarted)
func (se *Session) GetDuration() uint64 {
	if se.Status == SessionStatusCompleted {
		return uint64(se.Updated.Sub(se.Started).Seconds())
	} else {
		return 0
	}
}
