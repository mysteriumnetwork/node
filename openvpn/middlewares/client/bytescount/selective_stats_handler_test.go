/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"github.com/mysterium/node/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewSelectiveStatsHandlerEach(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	stats := SessionStats{
		BytesSent:     1,
		BytesReceived: 2,
	}

	handler, err := NewIntervalStatsHandler(statsRecorder.record, time.Now, time.Duration(0))
	assert.NoError(t, err)

	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
}

func TestNewSelectiveStatsHandlerEveryTheeSeconds(t *testing.T) {
	clock := utils.SettableClock{}
	statsRecorder := fakeStatsRecorder{}
	handler, _ := NewIntervalStatsHandler(statsRecorder.record, clock.GetTime, 3*time.Second)

	stats := SessionStats{
		BytesSent:     1,
		BytesReceived: 2,
	}
	emptyStats := SessionStats{}

	// first call executes handler
	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
	statsRecorder.LastSessionStats = emptyStats
	// call after 2s skips handler
	clock.AddTime(2 * time.Second)
	handler(stats)
	assert.Equal(t, emptyStats, statsRecorder.LastSessionStats)
	// call after 4s executes handler
	clock.AddTime(2 * time.Second)
	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
	statsRecorder.LastSessionStats = emptyStats

	// call after 1s skips handler
	clock.AddTime(1 * time.Second)
	handler(stats)
	assert.Equal(t, emptyStats, statsRecorder.LastSessionStats)
	// call after 30s executes handler
	clock.AddTime(29 * time.Second)
	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
}

func TestNewSelectiveStatsHandlerInvalidValues(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}

	_, err := NewIntervalStatsHandler(statsRecorder.record, time.Now, -1*time.Nanosecond)
	assert.EqualError(t, err, "Invalid 'interval' parameter")
}
