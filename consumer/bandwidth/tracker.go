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
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
)

const bitsInByte = 8

type publisher interface {
	Publish(topic string, data interface{})
}

// NewTracker creates instance of Tracker
func NewTracker(publisher publisher) *Tracker {
	return &Tracker{publisher: publisher}
}

// Tracker keeps track of current speed
type Tracker struct {
	publisher publisher

	previous connectionstate.Statistics
	lock     sync.RWMutex
}

// Subscribe subscribes to relevant events of event bus.
func (t *Tracker) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionSession, t.consumeSessionEvent); err != nil {
		return err
	}
	return bus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, t.consumeStatisticsEvent)
}

const consumeCooldown = 500 * time.Millisecond

// consumeStatisticsEvent handles the connection statistics changes
func (t *Tracker) consumeStatisticsEvent(evt connectionstate.AppEventConnectionStatistics) {
	t.lock.Lock()
	defer func() {
		t.lock.Unlock()
	}()

	// Skip speed calculation on the very first event.
	if t.previous.At.IsZero() {
		t.previous = evt.Stats
		return
	}

	secondsSince := evt.Stats.At.Sub(t.previous.At).Seconds()
	if secondsSince < consumeCooldown.Seconds() {
		log.Trace().Msgf("%fs passed since the last consumption, ignoring the event", secondsSince)
		return
	}

	byteDownDiff := evt.Stats.BytesReceived - t.previous.BytesReceived
	byteUpDiff := evt.Stats.BytesSent - t.previous.BytesSent

	t.publisher.Publish(AppTopicConnectionThroughput, AppEventConnectionThroughput{
		Throughput: Throughput{
			Up:   datasize.BitSpeed(float64(byteUpDiff) / secondsSince * bitsInByte),
			Down: datasize.BitSpeed(float64(byteDownDiff) / secondsSince * bitsInByte),
		},
		SessionInfo: evt.SessionInfo,
	})
	t.previous = evt.Stats
}

// consumeSessionEvent handles the session state changes
func (t *Tracker) consumeSessionEvent(sessionEvent connectionstate.AppEventConnectionSession) {
	t.lock.Lock()
	defer t.lock.Unlock()
	switch sessionEvent.Status {
	case connectionstate.SessionEndedStatus, connectionstate.SessionCreatedStatus:
		t.previous = connectionstate.Statistics{}
	}
}
