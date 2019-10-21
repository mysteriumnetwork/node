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
	"time"

	"github.com/rs/zerolog/log"
)

// Topic the topic that traversal events are published on
const Topic = "Traversal"

// Tracker is able to track NAT traversal events
type Tracker struct {
	lastEvent *Event
	eventChan chan Event
}

// BuildSuccessfulEvent returns new event for successful NAT traversal
func BuildSuccessfulEvent(stage string) Event {
	return Event{Stage: stage, Successful: true}
}

// BuildFailureEvent returns new event for failed NAT traversal
func BuildFailureEvent(stage string, err error) Event {
	return Event{Stage: stage, Successful: false, Error: err}
}

// NewTracker returns a new instance of event tracker
func NewTracker() *Tracker {
	return &Tracker{eventChan: make(chan Event, 1)}
}

// ConsumeNATEvent consumes a NAT event
func (et *Tracker) ConsumeNATEvent(event Event) {
	log.Info().Msgf("Got NAT event: %v", event)

	et.lastEvent = &event
	select {
	case et.eventChan <- event:
	case <-time.After(300 * time.Millisecond):
	}
}

// LastEvent returns the last known event and boolean flag, indicating if such event exists
func (et *Tracker) LastEvent() *Event {
	log.Info().Msgf("Getting last NAT event: %v", et.lastEvent)
	return et.lastEvent
}

// WaitForEvent waits for event to occur
func (et *Tracker) WaitForEvent() Event {
	if et.lastEvent != nil {
		return *et.lastEvent
	}
	e := <-et.eventChan
	log.Info().Msgf("Got NAT event: %v", e)
	return e
}

// Event represents a NAT traversal related event
type Event struct {
	Stage      string `json:"stage"`
	Successful bool   `json:"successful"`
	Error      error  `json:"error,omitempty"`
}
