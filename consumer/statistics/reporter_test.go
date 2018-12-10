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
	"errors"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	server_dto "github.com/mysteriumnetwork/node/server/dto"
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

func TestStatisticsReporterStartsAndStops(t *testing.T) {
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	mockSender := newMockRemoteSender()
	reporter := NewSessionStatisticsReporter(statisticsTracker, mockSender, mockSignerFactory, mockLocationDetector, time.Minute)

	reporter.ConsumeStateEvent(mockStateEvent)

	reporter.start(mockStateEvent.SessionInfo.ConsumerID, mockStateEvent.SessionInfo.Proposal.ServiceType, mockStateEvent.SessionInfo.Proposal.ProviderID, mockStateEvent.SessionInfo.SessionID)
	reporter.stop()

	assert.NoError(t, waitForChannel(mockSender.called, time.Millisecond*200))
	assert.False(t, reporter.started)
}

func TestStatisticsReporterInterval(t *testing.T) {
	mockSender := newMockRemoteSender()
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mockSender, mockSignerFactory, mockLocationDetector, time.Nanosecond)

	reporter.ConsumeStateEvent(mockStateEvent)

	reporter.start(mockStateEvent.SessionInfo.ConsumerID, mockStateEvent.SessionInfo.Proposal.ServiceType, mockStateEvent.SessionInfo.Proposal.ProviderID, mockStateEvent.SessionInfo.SessionID)
	assert.NoError(t, waitForChannel(mockSender.called, time.Millisecond*200))

	reporter.stop()
}

func TestStatisticsReporterConsumeStateEvent(t *testing.T) {
	mockSender := newMockRemoteSender()
	statisticsTracker := NewSessionStatisticsTracker(time.Now)
	reporter := NewSessionStatisticsReporter(statisticsTracker, mockSender, mockSignerFactory, mockLocationDetector, time.Nanosecond)
	reporter.ConsumeStateEvent(mockStateEvent)
	<-mockSender.called
	assert.True(t, reporter.started)
	copy := mockStateEvent
	copy.State = connection.Disconnecting
	reporter.ConsumeStateEvent(copy)
	assert.False(t, reporter.started)
}

func waitForChannel(ch chan bool, duration time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(duration):
		return errors.New("timed out waiting for channel")
	}
}

type mockRemoteSender struct {
	called chan bool
}

func (mrs *mockRemoteSender) SendSessionStats(id session.ID, stats server_dto.SessionStats, signer identity.Signer) error {
	mrs.called <- true
	return nil
}

func newMockRemoteSender() *mockRemoteSender {
	return &mockRemoteSender{
		called: make(chan bool),
	}
}

var _ RemoteReporter = &mockRemoteSender{}
