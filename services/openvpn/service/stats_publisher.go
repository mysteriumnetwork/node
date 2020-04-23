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
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/bytecount"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/session/event"
	"github.com/rs/zerolog/log"
)

func newStatsPublisher(clientMap *clientMap, bus eventbus.Publisher, frequencySeconds int) *statsPublisher {
	sb := &statsPublisher{
		clientMap: clientMap,
		bus:       bus,
	}
	sb.Middleware = bytecount.NewMiddleware(sb.handleStatsEvent, frequencySeconds)
	return sb
}

type statsPublisher struct {
	*bytecount.Middleware

	clientMap *clientMap
	bus       eventbus.Publisher
}

func (sb *statsPublisher) handleStatsEvent(clientStats bytecount.SessionByteCount) {
	sessions := sb.clientMap.GetClientSessions(clientStats.ClientID)
	if len(sessions) != 1 {
		log.Warn().Msgf(
			"Stats for unknown session. Expected a single session to exist for client %d, got %v sessions instead",
			clientStats.ClientID,
			len(sessions),
		)
		return
	}

	sb.bus.Publish(event.AppTopicDataTransferred, event.AppEventDataTransferred{
		ID:   string(sessions[0]),
		Up:   clientStats.BytesOut,
		Down: clientStats.BytesIn,
	})
}
