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

package statistics

import (
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
)

// TimeGetter function returns current time
type TimeGetter func() time.Time

// SessionStatisticsTracker keeps the session stats safe and sound
type SessionStatisticsTracker struct {
	sessionStats consumer.SessionStatistics
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatisticsTracker returns new session stats statisticsTracker with given timeGetter function
func NewSessionStatisticsTracker(timeGetter TimeGetter) *SessionStatisticsTracker {
	return &SessionStatisticsTracker{timeGetter: timeGetter}
}

// Retrieve retrieves session stats from statisticsTracker
func (sst *SessionStatisticsTracker) Retrieve() consumer.SessionStatistics {
	return sst.sessionStats
}

// MarkSessionStart marks current time as session start time for statistics
func (sst *SessionStatisticsTracker) markSessionStart() {
	time := sst.timeGetter()
	sst.sessionStart = &time
}

// GetSessionDuration returns elapsed time from marked session start
func (sst *SessionStatisticsTracker) GetSessionDuration() time.Duration {
	if sst.sessionStart == nil {
		return time.Duration(0)
	}
	duration := sst.timeGetter().Sub(*sst.sessionStart)
	return duration
}

// MarkSessionEnd stops counting session duration
func (sst *SessionStatisticsTracker) markSessionEnd() {
	sst.sessionStart = nil
}

// ConsumeStatisticsEvent handles the connection statistics changes
func (sst *SessionStatisticsTracker) ConsumeStatisticsEvent(stats consumer.SessionStatistics) {
	sst.sessionStats = stats
}

// ConsumeStateEvent handles the connection state changes
func (sst *SessionStatisticsTracker) ConsumeStateEvent(stateEvent connection.StateEvent) {
	switch stateEvent.State {
	case connection.Disconnecting:
		sst.markSessionEnd()
	case connection.Connected:
		sst.markSessionStart()
	}
}
