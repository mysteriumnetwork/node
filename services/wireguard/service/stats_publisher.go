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

package service

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/session/event"
)

type statsSupplier interface {
	PeerStats() (wgcfg.Stats, error)
}

type statsPublisher struct {
	done      chan struct{}
	bus       eventbus.Publisher
	frequency time.Duration
	once      sync.Once
}

func newStatsPublisher(bus eventbus.Publisher, frequency time.Duration) statsPublisher {
	return statsPublisher{
		done:      make(chan struct{}),
		bus:       bus,
		frequency: frequency,
	}
}

func (s *statsPublisher) start(sessionID string, supplier statsSupplier) {
	for {
		select {
		case <-time.After(s.frequency):
			stats, err := supplier.PeerStats()
			if err != nil {
				log.Warn().Err(err).Msg("Could not get peer statistics")
				continue
			}
			s.bus.Publish(event.AppTopicDataTransferred, event.AppEventDataTransferred{
				ID:   sessionID,
				Up:   stats.BytesSent,
				Down: stats.BytesReceived,
			})
		case <-s.done:
			log.Info().Msgf("Stopped publishing statistics for session %s", sessionID)
			return
		}
	}
}

func (s *statsPublisher) stop() {
	s.once.Do(func() {
		close(s.done)
	})
}
