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

package statistics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
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

func mockLocationDetector() location.Location {
	return location.Location{
		Country: "KG",
	}
}

func TestRemoteStatsSenderStartsAndStops(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	mysteriumClient := server.NewClient(ts.URL)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mysteriumClient, mockSignerFactory, mockLocationDetector, time.Minute)

	reporter.ConsumeStateEvent(mockStateEvent)

	reporter.start(mockStateEvent.SessionInfo.ConsumerID, mockStateEvent.SessionInfo.Proposal.ServiceType, mockStateEvent.SessionInfo.Proposal.ProviderID, mockStateEvent.SessionInfo.SessionID)
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 0 }))
	assert.True(t, reporter.started)
	reporter.stop()

	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) == 1 }))
	assert.False(t, reporter.started)
}

func TestRemoteStatsSenderInterval(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mysteriumClient, mockSignerFactory, mockLocationDetector, time.Nanosecond)

	reporter.ConsumeStateEvent(mockStateEvent)

	reporter.start(mockStateEvent.SessionInfo.ConsumerID, mockStateEvent.SessionInfo.Proposal.ServiceType, mockStateEvent.SessionInfo.Proposal.ProviderID, mockStateEvent.SessionInfo.SessionID)
	assert.NoError(t, waitFor(func() bool { return atomic.LoadInt64(&counter) > 3 }))

	reporter.stop()
}

func TestRemoteStatsStartsWithoutSigner(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mysteriumClient, mockSignerFactory, mockLocationDetector, time.Nanosecond)

	reporter.start(mockStateEvent.SessionInfo.ConsumerID, mockStateEvent.SessionInfo.Proposal.ServiceType, mockStateEvent.SessionInfo.Proposal.ProviderID, mockStateEvent.SessionInfo.SessionID)
	time.Sleep(time.Nanosecond * 2)
	assert.Equal(t, int64(0), atomic.LoadInt64(&counter))
	reporter.stop()
}

func TestRemoteStatsSenderConsumeStateEvent(t *testing.T) {
	var counter int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
	}))
	defer ts.Close()

	mysteriumClient := server.NewClient(ts.URL)
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mysteriumClient, mockSignerFactory, mockLocationDetector, time.Nanosecond)
	reporter.ConsumeStateEvent(mockStateEvent)
	assert.True(t, reporter.started)
	copy := mockStateEvent
	copy.State = connection.Disconnecting
	reporter.ConsumeStateEvent(copy)
	assert.False(t, reporter.started)
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
