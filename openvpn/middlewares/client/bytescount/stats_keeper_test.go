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

package bytescount

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/utils"
	"github.com/stretchr/testify/assert"
)

func TestStatsSavingWorks(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}

	statsKeeper.Save(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}

func TestGetSessionDurationReturnsFlooredDuration(t *testing.T) {
	settableClock := utils.SettableClock{}
	statsKeeper := NewSessionStatsKeeper(settableClock.GetTime)

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statsKeeper.MarkSessionStart()

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
	expectedDuration, err := time.ParseDuration("1s700000000ns")
	assert.NoError(t, err)
	duration := statsKeeper.GetSessionDuration()
	assert.Equal(t, expectedDuration, duration)
}

func TestGetSessionDurationFailsWhenSessionStartNotMarked(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)

	assert.Equal(t, time.Duration(0), statsKeeper.GetSessionDuration())
}

func TestStopSessionResetsSessionDuration(t *testing.T) {
	settableClock := utils.SettableClock{}
	statsKeeper := NewSessionStatsKeeper(settableClock.GetTime)

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statsKeeper.MarkSessionStart()

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
	statsKeeper.MarkSessionEnd()
	assert.Equal(t, time.Duration(0), statsKeeper.GetSessionDuration())
}
