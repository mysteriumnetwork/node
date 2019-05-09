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

package nat

import (
	"testing"

	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_StatusTracker_Status_ReturnsNotFinishedInitially(t *testing.T) {
	tracker := NewStatusTracker("last stage")
	status := tracker.Status()

	assert.Equal(t, "not_finished", status.Status)
	assert.Nil(t, status.Error)
}

func Test_StatusTracker_Status_ReturnsSuccessful_WithSuccessfulEvent(t *testing.T) {
	tracker := NewStatusTracker("last stage")
	tracker.ConsumeNATEvent(event.Event{Successful: true, Stage: "any stage"})
	status := tracker.Status()

	assert.Equal(t, "successful", status.Status)
	assert.Nil(t, status.Error)
}

func Test_StatusTracker_Status_ReturnsFailure_WithHolepunchingFailureEvent(t *testing.T) {
	tracker := NewStatusTracker("last stage")
	tracker.ConsumeNATEvent(event.Event{Successful: false, Stage: "last stage", Error: errors.New("test error")})
	status := tracker.Status()

	assert.Equal(t, "failure", status.Status)
	assert.EqualError(t, status.Error, "test error")
}

func Test_StatusTracker_Status_ReturnsNotFinished_WithPortMappingFailureEvent(t *testing.T) {
	tracker := NewStatusTracker("last stage")
	tracker.ConsumeNATEvent(event.Event{Successful: false, Stage: "first stage"})
	status := tracker.Status()

	assert.Equal(t, "not_finished", status.Status)
	assert.Nil(t, status.Error)
}

func Test_StatusTracker_Status_ReturnsNotFinished_AfterSuccess(t *testing.T) {
	tracker := NewStatusTracker("last stage")
	tracker.ConsumeNATEvent(event.Event{Successful: true, Stage: "any stage"})
	status := tracker.Status()

	assert.Equal(t, "successful", status.Status)
	assert.Nil(t, status.Error)

	tracker.ConsumeNATEvent(event.Event{Successful: false, Stage: "any stage"})
	status = tracker.Status()
	assert.Equal(t, "not_finished", status.Status)
}
