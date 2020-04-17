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

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
)

// Topic represents the different topics a consumer can subscribe to
const (
	// AppTopicConnectionState represents the session state change topic
	AppTopicConnectionState = "State"
	// AppTopicConnectionStatistics represents the session stats topic
	AppTopicConnectionStatistics = "Statistics"
	// AppTopicConnectionSession represents the session lifetime changes
	AppTopicConnectionSession = "Session"
)

// AppEventConnectionState is the struct we'll emit on a AppEventConnectionState topic event
type AppEventConnectionState struct {
	State       State
	SessionInfo Status
}

// State represents list of possible connection states
type State string

const (
	// NotConnected means no connection exists
	NotConnected = State("NotConnected")
	// Connecting means that connection is startCalled but not yet fully established
	Connecting = State("Connecting")
	// Connected means that fully established connection exists
	Connected = State("Connected")
	// Disconnecting means that connection close is in progress
	Disconnecting = State("Disconnecting")
	// Reconnecting means that connection is lost but underlying service is trying to reestablish it
	Reconnecting = State("Reconnecting")
	// Unknown means that we could not map the underlying transport state to our state
	Unknown = State("Unknown")
	// Canceled means that connection initialization was started, but failed never reaching Connected state
	Canceled = State("Canceled")
	// StateIPNotChanged means that consumer ip not changed after connection is created
	StateIPNotChanged = State("IPNotChanged")
	// StateConnectionFailed means that underlying connection is failed
	StateConnectionFailed = State("ConnectionFailed")
)

// Status holds connection state, session id and proposal of the connection
type Status struct {
	StartedAt    time.Time
	ConsumerID   identity.Identity
	AccountantID common.Address
	State        State
	SessionID    session.ID
	Proposal     market.ServiceProposal
}

// Duration returns elapsed time from marked session start
func (s *Status) Duration() time.Duration {
	if s.StartedAt.IsZero() {
		return time.Duration(0)
	}
	return time.Now().Sub(s.StartedAt)
}

const (
	// SessionCreatedStatus represents a session creation event
	SessionCreatedStatus = "Created"
	// SessionEndedStatus represents a session end
	SessionEndedStatus = "Ended"
)

// AppEventConnectionSession represents a session related event
type AppEventConnectionSession struct {
	Status      string
	SessionInfo Status
}

// AppEventConnectionStatistics represents a session statistics event
type AppEventConnectionStatistics struct {
	Stats       Statistics
	SessionInfo Status
}
