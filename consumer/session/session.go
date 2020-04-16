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
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/payments/crypto"
)

const (
	// SessionStatusNew means that newly created session object is written to storage
	SessionStatusNew = "New"
	// SessionStatusCompleted means that session object is updated on connection disconnect event
	SessionStatusCompleted = "Completed"
)

// History holds structure for saving session history
type History struct {
	SessionID       node_session.ID `storm:"id"`
	ConsumerID      identity.Identity
	AccountantID    string
	ProviderID      identity.Identity
	ServiceType     string
	ProviderCountry string
	Started         time.Time
	Status          string
	Updated         time.Time
	DataStats       connection.Statistics // is updated on disconnect event
	Invoice         crypto.Invoice        // is updated on disconnect event
}

// GetDuration returns delta in seconds (TimeUpdated - TimeStarted)
func (se *History) GetDuration() time.Duration {
	ended := se.Updated
	if ended.IsZero() {
		ended = time.Now()
	}
	return ended.Sub(se.Started)
}
