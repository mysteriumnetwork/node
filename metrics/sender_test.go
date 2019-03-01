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
	sentEvent    event
	mockResponse error
}

func buildMockEventsTransport(mockResponse error) *mockEventsTransport {
	return &mockEventsTransport{mockResponse: mockResponse}
}

func (transport *mockEventsTransport) sendEvent(event event) error {
	transport.sentEvent = event
	return transport.mockResponse
}

func TestSender_SendStartupEvent_SendsToTransport(t *testing.T) {
	mockTransport := buildMockEventsTransport(nil)
	sender := &Sender{Transport: mockTransport, ApplicationVersion: "test version"}

	err := sender.SendStartupEvent()
	assert.NoError(t, err)

	sentEvent := mockTransport.sentEvent

	assert.Equal(t, "startup", sentEvent.EventName)
	assert.Equal(t, applicationInfo{Name: "myst", Version: "test version"}, sentEvent.Application)
	assert.NotZero(t, sentEvent.CreatedAt)
}

func TestSender_SendStartupEvent_ReturnsTransportErrors(t *testing.T) {
	mockTransport := buildMockEventsTransport(nil)
	mockTransport.mockResponse = errors.New("mock error")
	sender := &Sender{Transport: mockTransport, ApplicationVersion: "test version"}

	err := sender.SendStartupEvent()
	assert.Error(t, err)
}

func TestSender_SendNATMappingResultEvent_SendsToTransport(t *testing.T) {
	mockTransport := buildMockEventsTransport(nil)
	sender := &Sender{Transport: mockTransport, ApplicationVersion: "test version"}

	err := sender.SendNATMappingResultEvent(true)
	assert.NoError(t, err)

	sentEvent := mockTransport.sentEvent
	assert.Equal(t, "nat_mapping_result", sentEvent.EventName)
	assert.Equal(t, applicationInfo{Name: "myst", Version: "test version"}, sentEvent.Application)
	assert.NotZero(t, sentEvent.CreatedAt)
	assert.Equal(t, natMappingResultContext{success: true}, sentEvent.Context)
}

func TestSender_SendNATMappingResultEvent_ReturnsTransportErrors(t *testing.T) {
	mockTransport := buildMockEventsTransport(nil)
	mockTransport.mockResponse = errors.New("mock error")
	sender := &Sender{Transport: mockTransport, ApplicationVersion: "test version"}

	error := sender.SendNATMappingResultEvent(true)
	assert.Error(t, error)
}
