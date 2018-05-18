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

package bytescount

import (
	"time"
)

// SessionStatsKeeper keeps session stats
type SessionStatsKeeper interface {
	Save(stats SessionStats)
	Retrieve() SessionStats
	MarkSessionStart()
	GetSessionDuration() time.Duration
	MarkSessionEnd()
}

// TimeGetter function returns current time
type TimeGetter func() time.Time

type sessionStatsKeeper struct {
	sessionStats SessionStats
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatsKeeper returns new session stats keeper with given timeGetter function
func NewSessionStatsKeeper(timeGetter TimeGetter) SessionStatsKeeper {
	return &sessionStatsKeeper{timeGetter: timeGetter}
}

// Save saves session stats to keeper
func (keeper *sessionStatsKeeper) Save(stats SessionStats) {
	keeper.sessionStats = stats
}

// Retrieve retrieves session stats from keeper
func (keeper *sessionStatsKeeper) Retrieve() SessionStats {
	return keeper.sessionStats
}

// MarkSessionStart marks current time as session start time for statistics
func (keeper *sessionStatsKeeper) MarkSessionStart() {
	time := keeper.timeGetter()
	keeper.sessionStart = &time
}

// GetSessionDuration returns elapsed time from marked session start
func (keeper *sessionStatsKeeper) GetSessionDuration() time.Duration {
	if keeper.sessionStart == nil {
		return time.Duration(0)
	}
	duration := keeper.timeGetter().Sub(*keeper.sessionStart)
	return duration
}

// MarkSessionEnd stops counting session duration
func (keeper *sessionStatsKeeper) MarkSessionEnd() {
	keeper.sessionStart = nil
}
