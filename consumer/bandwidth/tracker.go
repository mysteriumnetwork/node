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

package bandwidth

import (
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/rs/zerolog/log"
)

const bitsInByte = 8

// Throughput represents the throughput
type Throughput struct {
	BitsPerSecond float64
}

// String returns human readable form of the throughput
func (t Throughput) String() string {
	return bitCountDecimal(int64(t.BitsPerSecond))
}

// CurrentSpeed represents the current(moment) download and upload speeds in bits per second
type CurrentSpeed struct {
	Up, Down Throughput
}

// Tracker keeps track of current speed
type Tracker struct {
	previous     consumer.SessionStatistics
	previousTime time.Time
	currentSpeed CurrentSpeed
	lock         sync.RWMutex
}

// Get returns the current upload and download speeds in bits per second
func (t *Tracker) Get() CurrentSpeed {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.currentSpeed
}

// ConsumeStatisticsEvent handles the connection statistics changes
func (t *Tracker) ConsumeStatisticsEvent(stats consumer.SessionStatistics) {
	t.lock.Lock()
	defer func() {
		t.previous = stats
		t.lock.Unlock()
	}()

	if t.previousTime.IsZero() {
		t.previousTime = time.Now()
		return
	}

	currentTime := time.Now()
	secondsSince := currentTime.Sub(t.previousTime).Seconds()
	t.previousTime = currentTime

	byteDownDiff := stats.BytesReceived - t.previous.BytesReceived
	byteUpDiff := stats.BytesSent - t.previous.BytesSent

	t.currentSpeed = CurrentSpeed{
		Up:   Throughput{BitsPerSecond: float64(byteUpDiff) / secondsSince * bitsInByte},
		Down: Throughput{BitsPerSecond: float64(byteDownDiff) / secondsSince * bitsInByte},
	}

	log.Debug().Msgf("Download speed: %s", t.currentSpeed.Down)
	log.Debug().Msgf("Upload speed: %s", t.currentSpeed.Up)
}

// ConsumeSessionEvent handles the session state changes
func (t *Tracker) ConsumeSessionEvent(sessionEvent connection.SessionEvent) {
	t.lock.Lock()
	defer t.lock.Unlock()
	switch sessionEvent.Status {
	case connection.SessionEndedStatus, connection.SessionCreatedStatus:
		t.previous = consumer.SessionStatistics{}
		t.previousTime = time.Time{}
		t.currentSpeed = CurrentSpeed{}
	}
}

// bitCountDecimal returns a human readable representation of speed in bits per second
// Taken from: https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
func bitCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d bps", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cbps", float64(b)/float64(div), "kMGTPE"[exp])
}
