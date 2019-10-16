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

import "github.com/mysteriumnetwork/node/consumer"

// Topic represents the different topics a consumer can subscribe to.
const (
	// EventTopicState represents the connection state change topic.
	EventTopicState = "State"
	// EventTopicStatistics represents the connection stats topic.
	EventTopicStatistics = "Statistics"
	// EventTopicSession represents the session event.
	EventTopicSession = "Session"
)

// StateEvent is the struct we'll emit on a StateEvent topic event.
type StateEvent struct {
	State       State
	SessionInfo SessionInfo
}

// SessionStatus represents session status types.
type SessionStatus string

const (
	// SessionStatusCreated represents a session creation event.
	SessionStatusCreated = SessionStatus("Created")
	// SessionStatusEnded represents a session end.
	SessionStatusEnded = SessionStatus("Ended")
)

// SessionEvent represents a session related event.
type SessionEvent struct {
	Status      SessionStatus
	SessionInfo SessionInfo
}

// SessionStatsEvent represents a session stats event.
type SessionStatsEvent struct {
	Stats       consumer.SessionStatistics
	SessionInfo SessionInfo
}
