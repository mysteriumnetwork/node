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

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/eventbus"
)

type statsSupplier interface {
	Statistics() (connectionstate.Statistics, error)
}

type statsTracker struct {
	done     chan struct{}
	bus      eventbus.Publisher
	interval time.Duration

	mu        sync.RWMutex
	lastStats connectionstate.Statistics
}

func newStatsTracker(bus eventbus.Publisher, interval time.Duration) statsTracker {
	return statsTracker{
		done:     make(chan struct{}),
		bus:      bus,
		interval: interval,
	}
}

func (s *statsTracker) start(sessionSupplier *connectionManager, statsSupplier statsSupplier) {
	for {
		select {
		case <-time.After(s.interval):
			stats, err := statsSupplier.Statistics()
			if err != nil {
				log.Warn().Err(err).Msg("Could not get connection statistics")
				continue
			}

			s.bus.Publish(connectionstate.AppTopicConnectionStatistics, connectionstate.AppEventConnectionStatistics{
				Stats:       stats,
				UUID:        sessionSupplier.UUID(),
				SessionInfo: sessionSupplier.Status(),
			})

			s.mu.Lock()
			s.lastStats = stats
			s.mu.Unlock()

		case <-s.done:
			log.Info().Msg("Stopped publishing connection statistics")
			return
		}
	}
}

func (s *statsTracker) stats() connectionstate.Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastStats
}

func (s *statsTracker) stop() {
	s.done <- struct{}{}
}
