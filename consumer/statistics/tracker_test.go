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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/stretchr/testify/assert"
)

func TestStatsSavingWorks(t *testing.T) {
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	event := connection.SessionStatsEvent{
		Stats: consumer.SessionStatistics{BytesSent: 1, BytesReceived: 2},
	}

	statisticsTracker.ConsumeStatisticsEvent(event)
	assert.Equal(t, event.Stats, statisticsTracker.Retrieve())
}

func TestGetSessionDurationReturnsFlooredDuration(t *testing.T) {
	settableClock := utils.SettableClock{}
	statisticsTracker := NewSessionStatisticsTracker(settableClock.GetTime)

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statisticsTracker.markSessionStart()

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
	expectedDuration, err := time.ParseDuration("1s700000000ns")
	assert.NoError(t, err)
	duration := statisticsTracker.GetSessionDuration()
	assert.Equal(t, expectedDuration, duration)
}

func TestGetSessionDurationFailsWhenSessionStartNotMarked(t *testing.T) {
	statisticsTracker := NewSessionStatisticsTracker(time.Now)

	assert.Equal(t, time.Duration(0), statisticsTracker.GetSessionDuration())
}

func TestStopSessionResetsSessionDuration(t *testing.T) {
	settableClock := utils.SettableClock{}
	statisticsTracker := NewSessionStatisticsTracker(settableClock.GetTime)

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statisticsTracker.markSessionStart()

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
	statisticsTracker.markSessionEnd()
	assert.Equal(t, time.Duration(0), statisticsTracker.GetSessionDuration())
}

func TestStatisticsTrackerConsumeSessionEventCreated(t *testing.T) {
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	statisticsTracker.ConsumeSessionEvent(connection.SessionEvent{
		Status: connection.SessionStatusCreated,
	})
	assert.NotNil(t, statisticsTracker.sessionStart)
}

func TestStatisticsTrackerConsumeSessionEventEnded(t *testing.T) {
	now := time.Now()
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	statisticsTracker.sessionStart = &now
	statisticsTracker.ConsumeSessionEvent(connection.SessionEvent{
		Status: connection.SessionStatusEnded,
	})
	assert.Nil(t, statisticsTracker.sessionStart)
}

func TestConsumeStatisticsEventChain(t *testing.T) {
	sst := &SessionStatisticsTracker{
		timeGetter: time.Now,
	}
	statsEvent := connection.SessionStatsEvent{
		Stats: consumer.SessionStatistics{
			BytesReceived: 1,
			BytesSent:     1,
		},
	}
	sst.ConsumeStatisticsEvent(statsEvent)

	assert.EqualValues(t, statsEvent.Stats, sst.lastStats)
	assert.EqualValues(t, statsEvent.Stats, sst.sessionStats)

	sst.ConsumeStatisticsEvent(statsEvent)
	assert.EqualValues(t, statsEvent.Stats, sst.lastStats)
	assert.EqualValues(t, statsEvent.Stats, sst.sessionStats)

	updatedStatsEvent := connection.SessionStatsEvent{
		Stats: consumer.SessionStatistics{
			BytesReceived: 2,
			BytesSent:     2,
		},
	}

	sst.ConsumeStatisticsEvent(updatedStatsEvent)
	assert.EqualValues(t, updatedStatsEvent.Stats, sst.lastStats)
	assert.EqualValues(t, updatedStatsEvent.Stats, sst.sessionStats)

	statsAfterChain := consumer.SessionStatistics{
		BytesReceived: 3,
		BytesSent:     3,
	}

	// Simulate a reconnect now stats wise
	sst.ConsumeStatisticsEvent(statsEvent)
	assert.EqualValues(t, statsEvent.Stats, sst.lastStats)
	assert.EqualValues(t, statsAfterChain, sst.sessionStats)

	// Simulate no change in stats
	sst.ConsumeStatisticsEvent(statsEvent)
	assert.EqualValues(t, statsEvent.Stats, sst.lastStats)
	assert.EqualValues(t, statsAfterChain, sst.sessionStats)
}
