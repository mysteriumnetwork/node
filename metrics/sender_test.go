/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockEventsTransport struct {
	sentEvent    chan event
	mockResponse chan error
}

func buildMockEventsTransport() *mockEventsTransport {
	return &mockEventsTransport{sentEvent: make(chan event), mockResponse: make(chan error)}
}

func (transport *mockEventsTransport) sendEvent(event event) error {
	transport.sentEvent <- event
	return <-transport.mockResponse
}

func TestSender_SendStartupEvent_SendsEventWithoutBlocking(t *testing.T) {
	mockTransport := buildMockEventsTransport()
	sender := &Sender{Transport: mockTransport}

	sender.SendStartupEvent("test role", "test version")

	sentEvent := <-mockTransport.sentEvent
	mockTransport.mockResponse <- nil

	assert.Equal(t, "startup", sentEvent.EventName)
	assert.Equal(t, applicationInfo{Name: "myst", Version: "test version"}, sentEvent.Application)
	assert.Equal(t, startupContext{Role: "test role"}, sentEvent.Context)
	assert.NotZero(t, sentEvent.CreatedAt)
}

func TestSender_SendStartupEvent_IgnoresTransportErrors(t *testing.T) {
	mockTransport := buildMockEventsTransport()
	sender := &Sender{Transport: mockTransport}

	sender.SendStartupEvent("test role", "test version")

	sentEvent := <-mockTransport.sentEvent
	mockTransport.mockResponse <- errors.New("mock error")

	assert.NotNil(t, sentEvent)
}
