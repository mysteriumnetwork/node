/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package stats

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

var mockStateEvent = connection.StateEvent{
	State: connection.Connected,
	SessionInfo: connection.SessionInfo{
		ConsumerID: identity.FromAddress("0x000"),
		SessionID:  session.ID("test"),
		Proposal: dto.ServiceProposal{
			ServiceType: "just a test",
		},
	},
}

func mockSignerFactory(_ identity.Identity) identity.Signer {
	return &identity.SignerFake{}
}

func TestRemoteStatsSenderStartsAndStops(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	statsKeeper := NewSessionStatsKeeper(time.Now)
	mysteriumClient := server.NewClient(ts.URL)
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, mockSignerFactory, "KG", time.Minute)

	sender.ConsumeStateEvent(mockStateEvent)

	sender.start()
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 0 }))
	assert.True(t, sender.started)
	sender.stop()

	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 1 }))
	assert.False(t, sender.started)
}

func TestRemoteStatsSenderInterval(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statsKeeper := NewSessionStatsKeeper(time.Now)
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, mockSignerFactory, "KG", time.Nanosecond)

	sender.ConsumeStateEvent(mockStateEvent)

	sender.start()
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) > 3 }))

	sender.stop()
}

func TestRemoteStatsStartsWithoutSigner(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statsKeeper := NewSessionStatsKeeper(time.Now)
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, mockSignerFactory, "KG", time.Nanosecond)

	sender.start()
	time.Sleep(time.Nanosecond * 2)
	assert.Equal(t, int64(0), atomic.LoadInt64(&counter))
	sender.stop()
}

func TestRemoteStatsSenderConsumeStateEvent(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statsKeeper := NewSessionStatsKeeper(time.Now)
	sender := NewRemoteStatsSender(statsKeeper, mysteriumClient, mockSignerFactory, "KG", time.Nanosecond)
	sender.ConsumeStateEvent(mockStateEvent)
	assert.True(t, sender.started)
	copy := mockStateEvent
	copy.State = connection.Disconnecting
	sender.ConsumeStateEvent(copy)
	assert.False(t, sender.started)
}

func waitFor(f func() bool) error {
	timeout := time.Now().Add(time.Second)
	for time.Now().Before(timeout) {
		if f() {
			return nil
		}
	}
	return fmt.Errorf("Failed to wait for expected result")
}
