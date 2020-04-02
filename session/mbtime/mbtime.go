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

package mbtime

import (
	"syscall"
	"time"

	// Needed for go:linkname
	_ "unsafe"

	"github.com/rs/zerolog/log"
)

// Time represents structure for calculating elapsed duration between time interval.
// It uses different syscalls depending on platform expect for Windows it uses std
// library time since it already includes boottime.
type Time struct {
	// ns represents suspend-aware monotonic clock time in nanoseconds.
	ns time.Duration
}

// Now returns current suspend-aware monotonic time.
func Now() Time {
	ts, err := nanotime()
	if err != nil {
		log.Error().Err(err).Msgf("Using fallback time")
		return Time{ns: time.Duration(nanotimeFallback())}
	}
	return Time{ns: time.Duration(ts)}
}

// New creates new suspend-aware monotonic time from given initial values.
// This is mostly useful for unit tests.
func New(sec int64, nsec int64) Time {
	return Time{ns: time.Duration(sec*1e9 + nsec)}
}

// Nano returns current duration in nanoseconds.
func (t Time) Nano() time.Duration {
	return t.ns
}

const (
	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1
)

// Sub returns the duration t-u.
func (t Time) Sub(u Time) time.Duration {
	d := t.ns - u.ns
	if d < 0 && t.ns > u.ns {
		return maxDuration // t - u is positive out of range
	}
	if d > 0 && t.ns < u.ns {
		return minDuration // t - u is negative out of range
	}
	return d
}

// String returns time formatted time string.
func (t Time) String() string {
	return t.ns.String()
}

// Since returns the time elapsed since t.
// It is shorthand for time.Now().Sub(t).
func Since(u Time) time.Duration {
	return Now().ns - u.ns
}

func tempSyscallErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}

//go:noescape
//go:linkname nanotimeFallback runtime.nanotime
func nanotimeFallback() int64
