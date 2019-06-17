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

package event

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
	gatewaysSent  []map[string]string
}

func buildMockMetricsSender(mockResponse error) *mockMetricsSender {
	return &mockMetricsSender{mockResponse: mockResponse, successSent: false}
}

func (sender *mockMetricsSender) SendNATMappingSuccessEvent(stage string, gateways []map[string]string) {
	sender.successSent = true
	sender.stageSent = stage
	sender.gatewaysSent = gateways
}

func (sender *mockMetricsSender) SendNATMappingFailEvent(stage string, gateways []map[string]string, err error) {
	sender.failErrorSent = err
	sender.stageSent = stage
	sender.gatewaysSent = gateways
}

type mockIPResolver struct {
	mockIp  string
	mockErr error
}

func (resolver *mockIPResolver) GetPublicIP() (string, error) {
	return resolver.mockIp, resolver.mockErr
}

var (
	mockGateways = []map[string]string{
		{"test": "test"},
	}
	mockGatewayLoader = func() []map[string]string {
		return mockGateways
	}
)

func Test_EventsSender_ConsumeNATEvent_SendsSuccessEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	sender.ConsumeNATEvent(Event{Stage: "hole_punching", Successful: true})

	assert.True(t, mockMetricsSender.successSent)
	assert.Equal(t, "hole_punching", mockMetricsSender.stageSent)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_WithSameIp_DoesNotSendSuccessEventAgain(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	sender.ConsumeNATEvent(Event{Successful: true})

	mockMetricsSender.successSent = false

	sender.ConsumeNATEvent(Event{Successful: true})
	assert.False(t, mockMetricsSender.successSent)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_WithDifferentIP_SendsSuccessEventAgain(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	sender.ConsumeNATEvent(Event{Successful: true})

	mockMetricsSender.successSent = false

	mockIPResolver.mockIp = "2nd ip"
	sender.ConsumeNATEvent(Event{Successful: true})
	assert.True(t, mockMetricsSender.successSent)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_WhenIPResolverFails_DoesNotSendEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockErr: errors.New("mock error")}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	sender.ConsumeNATEvent(Event{Successful: true})

	assert.False(t, mockMetricsSender.successSent)
	assert.Nil(t, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_SendsFailureEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	testErr := errors.New("test error")
	sender.ConsumeNATEvent(Event{Stage: "hole_punching", Successful: false, Error: testErr})

	assert.Equal(t, testErr, mockMetricsSender.failErrorSent)
	assert.Equal(t, "hole_punching", mockMetricsSender.stageSent)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_WithFailuresOfDifferentStages_SendsBothEvents(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	testErr1 := errors.New("test error 1")
	sender.ConsumeNATEvent(Event{Successful: false, Error: testErr1, Stage: "test 1"})
	testErr2 := errors.New("test error 2")
	sender.ConsumeNATEvent(Event{Successful: false, Error: testErr2, Stage: "test 2"})

	assert.Equal(t, testErr2, mockMetricsSender.failErrorSent)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}

func Test_EventsSender_ConsumeNATEvent_WithSuccessAndFailureOnSameIp_SendsBothEvents(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	mockIPResolver := &mockIPResolver{mockIp: "1st ip"}
	sender := NewSender(mockMetricsSender, mockIPResolver.GetPublicIP, mockGatewayLoader)

	sender.ConsumeNATEvent(Event{Successful: true})
	testErr := errors.New("test error")
	sender.ConsumeNATEvent(Event{Successful: false, Error: testErr})

	assert.True(t, mockMetricsSender.successSent)
	assert.Equal(t, mockMetricsSender.failErrorSent, testErr)
	assert.Equal(t, mockGateways, mockMetricsSender.gatewaysSent)
}
