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
	mockResponse  error
	successSent   bool
	failErrorSent error
}

func buildMockMetricsSender(mockResponse error) *mockMetricsSender {
	return &mockMetricsSender{mockResponse: mockResponse, successSent: false}
}

func (sender *mockMetricsSender) SendNATMappingSuccessEvent() error {
	sender.successSent = true
	return nil
}

func (sender *mockMetricsSender) SendNATMappingFailEvent(err error) error {
	sender.failErrorSent = err
	return nil
}

func Test_EventsSender_ConsumerNATEvent_SendsSuccessEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	sender := NewEventsSender(mockMetricsSender)

	sender.ConsumeNATEvent(Event{Type: SuccessEventType})

	assert.True(t, mockMetricsSender.successSent)
}

func Test_EventsSender_ConsumerNATEvent_SendsFailureEvent(t *testing.T) {
	mockMetricsSender := buildMockMetricsSender(nil)
	sender := NewEventsSender(mockMetricsSender)

	mockErr := errors.New("test error")
	sender.ConsumeNATEvent(Event{Type: FailureEventType, Error: mockErr})

	assert.Equal(t, mockErr, mockMetricsSender.failErrorSent)
}
