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
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

const (
	// StatusNew means that newly created session object is written to storage
	StatusNew = "New"
	// StatusCompleted means that session object is updated on connection disconnect event
	StatusCompleted = "Completed"
)

const (
	// DirectionConsumed marks traffic transaction where node participated as consumer.
	DirectionConsumed = "Consumed"
	// DirectionProvided marks traffic transaction where node participated as provider.
	DirectionProvided = "Provided"
)

// History holds structure for saving session history
type History struct {
	SessionID       node_session.ID `storm:"id"`
	Direction       string
	ConsumerID      identity.Identity
	HermesID        string
	ProviderID      identity.Identity
	ServiceType     string
	ConsumerCountry string
	ProviderCountry string
	DataSent        uint64
	DataReceived    uint64
	Tokens          *big.Int

	IPType string

	Status  string
	Started time.Time
	Updated time.Time
}

// GetDuration returns delta in seconds (TimeUpdated - TimeStarted)
func (se *History) GetDuration() time.Duration {
	ended := se.Updated
	if ended.IsZero() {
		ended = time.Now()
	}
	return ended.Sub(se.Started)
}
