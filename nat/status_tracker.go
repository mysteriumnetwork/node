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
	// StatusNotFinished describes unknown NAT status assigned before any checks done.
	StatusNotFinished = "not_finished"
	// StatusSuccessful describes success NAT status assigned when NAT traversal succeeded.
	StatusSuccessful = "successful"
	// StatusFailure describes failed NAT status assigned when NAT traversal failed.
	StatusFailure = "failure"
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
	if event.Stage == t.lastStageName && !event.Successful {
		t.status = Status{Status: StatusFailure, Error: event.Error}
		return
	}

	if event.Successful {
		t.status = Status{Status: StatusSuccessful}
		return
	}

	t.status = Status{Status: StatusNotFinished}
}

// NewStatusTracker returns new instance of status tracker
// TODO check with Dmitri if this is needed as we have NodeStatus now
func NewStatusTracker(lastStageName string) *StatusTracker {
	return &StatusTracker{
		lastStageName: lastStageName,
		status:        Status{Status: StatusNotFinished},
	}
}
