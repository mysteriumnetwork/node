/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package connection

import "time"

// Statistics represents connection statistics.
type Statistics struct {
	At                       time.Time
	BytesSent, BytesReceived uint64
}

// Diff calculates the difference in bytes between the old stats and new.
func (stats Statistics) Diff(new Statistics) Statistics {
	return Statistics{
		At:            new.At,
		BytesSent:     diff(stats.BytesSent, new.BytesSent),
		BytesReceived: diff(stats.BytesReceived, new.BytesReceived),
	}
}

// diff takes in the old and the new values of statistics, returns the calculated delta.
func diff(old, new uint64) (res uint64) {
	if old > new {
		return new
	}
	return new - old
}

// Plus adds up the given statistics with the diff and returns new stats
func (stats Statistics) Plus(diff Statistics) Statistics {
	return Statistics{
		At:            stats.At,
		BytesReceived: stats.BytesReceived + diff.BytesReceived,
		BytesSent:     stats.BytesSent + diff.BytesSent,
	}
}
