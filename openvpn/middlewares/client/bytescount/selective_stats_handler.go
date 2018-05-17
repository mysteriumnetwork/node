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
	"errors"
	"time"
)

// NewIntervalStatsHandler creates and returns composite handler, which invokes internal handler at given interval
func NewIntervalStatsHandler(handler SessionStatsHandler, currentTime func() time.Time, interval time.Duration) (SessionStatsHandler, error) {
	if interval < 0 {
		return nil, errors.New("Invalid 'interval' parameter")
	}

	firstTime := true
	var lastTime time.Time
	return func(sessionStats SessionStats) error {
		now := currentTime()
		if firstTime || (now.Sub(lastTime)) >= interval {
			firstTime = false
			lastTime = now
			return handler(sessionStats)
		}
		return nil
	}, nil
}
