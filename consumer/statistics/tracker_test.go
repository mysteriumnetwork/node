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
	"reflect"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/stretchr/testify/assert"
)

func TestStatsSavingWorks(t *testing.T) {
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	stats := consumer.SessionStatistics{BytesSent: 1, BytesReceived: 2}

	statisticsTracker.ConsumeStatisticsEvent(stats)
	assert.Equal(t, stats, statisticsTracker.Retrieve())
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

func TestSessionStatisticsTracker_GetStatisticsDiff(t *testing.T) {
	exampleStats := consumer.SessionStatistics{
		BytesReceived: 1,
		BytesSent:     2,
	}
	type args struct {
		old consumer.SessionStatistics
		new consumer.SessionStatistics
	}
	tests := []struct {
		name string
		args args
		want consumer.SessionStatistics
	}{
		{
			name: "calculates statistics correctly if they are continuous",
			args: args{
				old: consumer.SessionStatistics{},
				new: exampleStats,
			},
			want: exampleStats,
		},
		{
			name: "calculates statistics correctly if they are not continuous",
			args: args{
				old: consumer.SessionStatistics{
					BytesReceived: 5,
					BytesSent:     6,
				},
				new: exampleStats,
			},
			want: exampleStats,
		},
		{
			name: "returns zeros on no change",
			args: args{
				old: exampleStats,
				new: exampleStats,
			},
			want: consumer.SessionStatistics{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sst := &SessionStatisticsTracker{
				timeGetter: time.Now,
			}
			if got := sst.GetStatisticsDiff(tt.args.old, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SessionStatisticsTracker.GetStatisticsDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsumeStatisticsEventChain(t *testing.T) {
	sst := &SessionStatisticsTracker{
		timeGetter: time.Now,
	}
	stats := consumer.SessionStatistics{
		BytesReceived: 1,
		BytesSent:     1,
	}
	sst.ConsumeStatisticsEvent(stats)

	assert.EqualValues(t, stats, sst.lastStats)
	assert.EqualValues(t, stats, sst.sessionStats)

	sst.ConsumeStatisticsEvent(stats)
	assert.EqualValues(t, stats, sst.lastStats)
	assert.EqualValues(t, stats, sst.sessionStats)

	updatedStats := consumer.SessionStatistics{
		BytesReceived: 2,
		BytesSent:     2,
	}

	sst.ConsumeStatisticsEvent(updatedStats)
	assert.EqualValues(t, updatedStats, sst.lastStats)
	assert.EqualValues(t, updatedStats, sst.sessionStats)

	statsAfterChain := consumer.SessionStatistics{
		BytesReceived: 3,
		BytesSent:     3,
	}

	// Simulate a reconnect now stats wise
	sst.ConsumeStatisticsEvent(stats)
	assert.EqualValues(t, stats, sst.lastStats)
	assert.EqualValues(t, statsAfterChain, sst.sessionStats)

	// Simulate no change in stats
	sst.ConsumeStatisticsEvent(stats)
	assert.EqualValues(t, stats, sst.lastStats)
	assert.EqualValues(t, statsAfterChain, sst.sessionStats)
}
