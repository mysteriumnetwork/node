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

package traversal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockMetricsSender struct {
	mockResponse error

	successSent   bool
	failErrorSent error
	stageSent     string
}

func buildMockMetricsSender(mockResponse error) *mockMetricsSender {
	return &mockMetricsSender{mockResponse: mockResponse, successSent: false}
}

func (sender *mockMetricsSender) SendNATMappingSuccessEvent(stage string) error {
	sender.successSent = true
	sender.stageSent = stage
	return nil
}

func (sender *mockMetricsSender) SendNATMappingFailEvent(stage string, err error) error {
	sender.failErrorSent = err
	sender.stageSent = stage
	return nil
}

type mockIPResolver struct {
	mockIp  string
	mockErr error
}

func (resolver *mockIPResolver) GetPublicIP() (string, error) {
	return resolver.mockIp, resolver.mockErr
}

func Test_EventsSender_ConsumeNATEvent_SendsSuccessEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	sender.ConsumeNATEvent(Event{Stage: "hole_punching", Type: SuccessEventType})

	assert.True(t, mockMetricsSender.successSent)
	assert.Equal(t, "hole_punching", mockMetricsSender.stageSent)
}

func Test_EventsSender_ConsumeNATEvent_WithSameIp_DoesNotSendSuccessEventAgain(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})

	mockMetricsSender.successSent = false

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})
	assert.False(t, mockMetricsSender.successSent)
}

func Test_EventsSender_ConsumeNATEvent_WithDifferentIP_SendsSuccessEventAgain(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockIp: "1st ip"}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})

	mockMetricsSender.successSent = false

	mockIPResolver.mockIp = "2nd ip"
	sender.ConsumeNATEvent(Event{Type: SuccessEventType})
	assert.True(t, mockMetricsSender.successSent)
}

func Test_EventsSender_ConsumeNATEvent_WhenIPResolverFails_DoesNotSendEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockErr: errors.New("mock error")}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})

	assert.False(t, mockMetricsSender.successSent)
}

func Test_EventsSender_ConsumeNATEvent_SendsFailureEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	testErr := errors.New("test error")
	sender.ConsumeNATEvent(Event{Stage: "hole_punching", Type: FailureEventType, Error: testErr})

	assert.Equal(t, testErr, mockMetricsSender.failErrorSent)
	assert.Equal(t, "hole_punching", mockMetricsSender.stageSent)
}

func Test_EventsSender_ConsumeNATEvent_WithSuccessAndFailureOnSameIp_SendsBothEvents(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockIp: "1st ip"}
	sender := NewEventsSender(mockMetricsSender, mockIPResolver.GetPublicIP)

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})
	testErr := errors.New("test error")
	sender.ConsumeNATEvent(Event{Type: FailureEventType, Error: testErr})

	assert.True(t, mockMetricsSender.successSent)
	assert.Equal(t, mockMetricsSender.failErrorSent, testErr)
}
