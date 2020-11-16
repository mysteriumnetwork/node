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

package broker

import (
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/session/event"
	"github.com/rs/zerolog/log"
)

type statsPublisher struct {
	done      chan struct{}
	bus       eventbus.Publisher
	frequency time.Duration
}

type statsSupplier interface {
	stats() (rx, tx uint64)
}

func newStatsPublisher(bus eventbus.Publisher, frequency time.Duration) statsPublisher {
	return statsPublisher{
		done:      make(chan struct{}),
		bus:       bus,
		frequency: frequency,
	}
}

func (s statsPublisher) start(sessionID string, supplier statsSupplier) {
	for {
		select {
		case <-time.After(s.frequency):
			rx, tx := supplier.stats()

			s.bus.Publish(event.AppTopicDataTransferred, event.AppEventDataTransferred{
				ID:   sessionID,
				Up:   tx,
				Down: rx,
			})
		case <-s.done:
			log.Info().Msgf("Stopped publishing statistics for session %s", sessionID)
			return
		}
	}
}

func (s statsPublisher) stop() {
	close(s.done)
}
