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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

type fakeSupplier struct {
}

func (f fakeSupplier) PeerStats() (*wireguard.Stats, error) {
	return &wireguard.Stats{
		BytesSent:     25,
		BytesReceived: 52,
		LastHandshake: time.Now(),
	}, nil
}

func Test_statsPublisher_start(t *testing.T) {
	bus := mocks.NewEventBus()
	publisher := newStatsPublisher(bus, 5*time.Millisecond)

	go publisher.start("kappa", &fakeSupplier{})

	assert.Eventually(t, func() bool {
		lastEvt := bus.Pop()
		if lastEvt == nil {
			return false
		}
		evt, ok := lastEvt.(event.DataTransferEventPayload)
		assert.True(t, ok)
		return evt.ID == "kappa" && evt.Down == 52 && evt.Up == 25
	}, 1*time.Second, 5*time.Millisecond)

	publisher.stop()

	assert.Never(t, func() bool {
		return bus.Pop() != nil
	}, 1*time.Second, 5*time.Millisecond)
}
