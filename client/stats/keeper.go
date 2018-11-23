/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package stats

import (
	"time"

	"github.com/mysteriumnetwork/node/client"
	"github.com/mysteriumnetwork/node/core/connection"

	"github.com/mysteriumnetwork/node/client/stats/dto"
)

// TimeGetter function returns current time
type TimeGetter func() time.Time

// SessionStatsKeeper keeps the session stats safe and sound
type SessionStatsKeeper struct {
	sessionStats dto.SessionStats
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatsKeeper returns new session stats keeper with given timeGetter function
func NewSessionStatsKeeper(timeGetter TimeGetter) *SessionStatsKeeper {
	return &SessionStatsKeeper{timeGetter: timeGetter}
}

// Retrieve retrieves session stats from keeper
func (keeper *SessionStatsKeeper) Retrieve() dto.SessionStats {
	return keeper.sessionStats
}

// MarkSessionStart marks current time as session start time for statistics
func (keeper *SessionStatsKeeper) markSessionStart() {
	time := keeper.timeGetter()
	keeper.sessionStart = &time
}

// GetSessionDuration returns elapsed time from marked session start
func (keeper *SessionStatsKeeper) GetSessionDuration() time.Duration {
	if keeper.sessionStart == nil {
		return time.Duration(0)
	}
	duration := keeper.timeGetter().Sub(*keeper.sessionStart)
	return duration
}

// MarkSessionEnd stops counting session duration
func (keeper *SessionStatsKeeper) markSessionEnd() {
	keeper.sessionStart = nil
}

// Subscribe subscribes the keeper on the bus for relevant events
func (keeper *SessionStatsKeeper) Subscribe(bus client.EventSubscriptionKeeper) {
	bus.Subscribe(string(connection.StatsEvent), keeper.consumeStatsEvent)
	bus.Subscribe(string(connection.StateEvent), keeper.consumeStateEvent)
}

// Unsubscribe unsubscribes the sender from bus
func (keeper *SessionStatsKeeper) Unsubscribe(bus client.EventSubscriptionKeeper) {
	bus.Unsubscribe(string(connection.StatsEvent), keeper.consumeStatsEvent)
	bus.Unsubscribe(string(connection.StateEvent), keeper.consumeStateEvent)
}

func (keeper *SessionStatsKeeper) consumeStatsEvent(stats dto.SessionStats) {
	keeper.sessionStats = stats
}

func (keeper *SessionStatsKeeper) consumeStateEvent(stateEvent connection.StateEventPayload) {
	switch stateEvent.State {
	case connection.Disconnecting:
		keeper.markSessionEnd()
	case connection.Connected:
		keeper.markSessionStart()
	}
}
