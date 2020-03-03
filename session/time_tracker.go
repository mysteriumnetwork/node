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
	"time"

	"github.com/mysteriumnetwork/node/session/mbtime"
)

// TimeTracker tracks elapsed time from the beginning of the session
// it's passive (no internal go routines) and simply encapsulates time operation: now - startOfSession expressed as duration
type TimeTracker struct {
	started   bool
	startTime mbtime.Time
	getTime   func() mbtime.Time
}

// NewTracker initializes TimeTracker with specified monotonically increasing clock function (usually time.Now is enough - but we do DI for test sake)
func NewTracker(getTime func() mbtime.Time) TimeTracker {
	return TimeTracker{
		getTime: getTime,
	}
}

// StartTracking starts tracking the time
func (tt *TimeTracker) StartTracking() {
	tt.started = true
	tt.startTime = tt.getTime()
}

// Elapsed gets the total duration of time that has passed since we've started
func (tt TimeTracker) Elapsed() time.Duration {
	if !tt.started {
		return 0 * time.Second
	}
	t := tt.getTime()
	return t.Sub(tt.startTime)
}
