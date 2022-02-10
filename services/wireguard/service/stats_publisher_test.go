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

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/session/event"
)

type fakeSupplier struct{}

func (f fakeSupplier) PeerStats() (wgcfg.Stats, error) {
	return wgcfg.Stats{
		BytesSent:     25,
		BytesReceived: 52,
		LastHandshake: time.Now(),
	}, nil
}

func Test_statsPublisher_start(t *testing.T) {
	bus := mocks.NewEventBus()
	publisher := newStatsPublisher(bus, time.Microsecond)

	go publisher.start("kappa", &fakeSupplier{})

	assert.Eventually(t, func() bool {
		lastEvt := bus.Pop()
		if lastEvt == nil {
			return false
		}
		evt, ok := lastEvt.(event.AppEventDataTransferred)
		assert.True(t, ok)
		return evt.ID == "kappa" && evt.Down == 52 && evt.Up == 25
	}, 2*time.Second, 10*time.Millisecond)

	publisher.stop()

	// After stop publisher may still publish one last event if it was started before stop
	// so we need to wait and drain last event from bus before checking if publisher
	// actually stopped publishing events.
	time.Sleep(time.Millisecond)
	bus.Pop()
	assert.Never(t, func() bool {
		return bus.Pop() != nil
	}, time.Millisecond, time.Microsecond)
}
