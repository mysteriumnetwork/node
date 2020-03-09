/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package session

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/stretchr/testify/assert"
)

func Test_NotStartedTrackerElapsedReturnsZeroValue(t *testing.T) {
	tt := NewTracker(func() mbtime.Time { return mbtime.New(10, 10) })

	assert.Equal(t, 0*time.Second, tt.Elapsed())
}

func Test_ElapsedReturnsCorrectValue(t *testing.T) {
	mockedClock := newMockedTime(
		[]mbtime.Time{
			mbtime.New(1, 0),
			mbtime.New(4, 0),
		},
	)
	tt := NewTracker(mockedClock)

	tt.StartTracking()
	elapsed := tt.Elapsed()

	assert.Equal(t, 3*time.Second, elapsed)
}

func newMockedTime(timeValues []mbtime.Time) func() mbtime.Time {
	count := 0

	return func() mbtime.Time {
		val := timeValues[count]
		count = count + 1
		return val
	}
}
