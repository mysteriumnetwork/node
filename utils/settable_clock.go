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

package utils

import "time"

// SettableClock allows settings and getting time, which is convenient for testing
type SettableClock struct {
	time time.Time
}

// SetTime sets time to be returned from GetTime
func (clock *SettableClock) SetTime(time time.Time) {
	clock.time = time
}

// GetTime returns set time
func (clock *SettableClock) GetTime() time.Time {
	return clock.time
}

// AddTime adds given duration to current clock time
func (clock *SettableClock) AddTime(duration time.Duration) {
	newTime := clock.GetTime().Add(duration)
	clock.SetTime(newTime)
}
