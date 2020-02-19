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
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

// StatsReportInterval is interval for consumer connection statistics reporting.
const StatsReportInterval = 2 * time.Second

type statsSupplier interface {
	Statistics() (Statistics, error)
}

type statsPublisher struct {
	done     chan struct{}
	bus      eventbus.Publisher
	interval time.Duration
}

func newStatsPublisher(bus eventbus.Publisher) statsPublisher {
	return statsPublisher{
		done:     make(chan struct{}),
		bus:      bus,
		interval: StatsReportInterval,
	}
}

func (s statsPublisher) start(sessionInfo SessionInfo, supplier statsSupplier) {
	for {
		select {
		case <-time.After(s.interval):
			stats, err := supplier.Statistics()
			if err != nil {
				log.Warn().Err(err).Msg("Could not get peer statistics")
				continue
			}
			s.bus.Publish(AppTopicConsumerStatistics, SessionStatsEvent{
				Stats:       stats,
				SessionInfo: sessionInfo,
			})
		case <-s.done:
			log.Info().Msgf("Stopped publishing statistics for session %s", sessionInfo.SessionID)
			return
		}
	}
}

func (s statsPublisher) stop() {
	s.done <- struct{}{}
}
