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
	"github.com/rs/zerolog/log"
)

// TimeGetter function returns current time
type TimeGetter func() time.Time

// SessionStatisticsTracker keeps the session stats safe and sound
type SessionStatisticsTracker struct {
	lastStats    connection.Statistics
	sessionStats connection.Statistics
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatisticsTracker returns new session stats statisticsTracker with given timeGetter function
func NewSessionStatisticsTracker(timeGetter TimeGetter) *SessionStatisticsTracker {
	return &SessionStatisticsTracker{timeGetter: timeGetter}
}

// Retrieve retrieves session stats from statisticsTracker
func (sst *SessionStatisticsTracker) Retrieve() connection.Statistics {
	return sst.sessionStats
}

// Reset resets session stats to 0
func (sst *SessionStatisticsTracker) Reset() {
	sst.sessionStats = connection.Statistics{}
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
func (sst *SessionStatisticsTracker) ConsumeStatisticsEvent(e connection.SessionStatsEvent) {
	sst.sessionStats = sst.sessionStats.Plus(sst.lastStats.Diff(e.Stats))
	sst.lastStats = e.Stats
	log.Debug().Msgf("bytes received %v, sent %v", consumer.BitCountDecimal(sst.sessionStats.BytesReceived, "B"), consumer.BitCountDecimal(sst.sessionStats.BytesSent, "B"))
}

// ConsumeSessionEvent handles the session state changes
func (sst *SessionStatisticsTracker) ConsumeSessionEvent(sessionEvent connection.SessionEvent) {
	switch sessionEvent.Status {
	case connection.SessionEndedStatus:
		sst.markSessionEnd()
	case connection.SessionCreatedStatus:
		sst.markSessionStart()
	}
}
