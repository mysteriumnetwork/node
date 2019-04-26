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
	"github.com/mysteriumnetwork/node/nat/event"
)

// StatusTracker keeps status of NAT traversal by consuming NAT events - whether if finished and was it successful.
// It can finish either by successful event from any stage, or by a failure of the last stage.
type StatusTracker struct {
	lastStageName string
	status        Status
}

const (
	statusNotFinished = "not_finished"
	statusSuccessful  = "successful"
	statusFailure     = "failure"
)

// Status represents NAT traversal status (either "not_finished", "successful" or "failure") and an optional error.
type Status struct {
	Status string
	Error  error
}

// Status returns NAT traversal status
func (t *StatusTracker) Status() Status {
	return t.status
}

// ConsumeNATEvent processes NAT event to determine NAT traversal status
func (t *StatusTracker) ConsumeNATEvent(event event.Event) {
	if event.Stage == t.lastStageName && event.Successful == false {
		t.status = Status{Status: statusFailure, Error: event.Error}
		return
	}

	if event.Successful {
		t.status = Status{Status: statusSuccessful}
		return
	}
}

// NewStatusTracker returns new instance of status tracker
func NewStatusTracker(lastStageName string) *StatusTracker {
	return &StatusTracker{
		lastStageName: lastStageName,
		status:        Status{Status: statusNotFinished},
	}
}
