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
	"time"

	log "github.com/cihub/seelog"
)

// EventTopic the topic that traversal events are published on
const EventTopic = "Traversal"

// EventNoEvent represents an occurence where nothing really happened
var EventNoEvent = Event{Name: ""}

// EventSuccess represents a succesful NAT traversal
var EventSuccess = Event{Name: "success"}

// EventFailure represents a failed NAT traversal event
var EventFailure = Event{Name: "failure"}

// EventsTracker is able to track nat traversal events
type EventsTracker struct {
	lastEvent Event
	eventChan chan Event
}

// NewEventsTracker returns a new instance of event tracker
func NewEventsTracker() *EventsTracker {
	return &EventsTracker{eventChan: make(chan Event, 1)}
}

// ConsumeNATEvent consumes a NAT event
func (et *EventsTracker) ConsumeNATEvent(event Event) {
	log.Info("got NAT event: ", event)
	et.lastEvent = event
	select {
	case et.eventChan <- event:
	case <-time.After(300 * time.Millisecond):
	}
}

// LastEvent returns the last known event
func (et *EventsTracker) LastEvent() Event {
	log.Info("getting last NAT event: ", et.lastEvent)
	return et.lastEvent
}

// WaitForEvent waits for event to occur
func (et *EventsTracker) WaitForEvent() Event {
	if et.lastEvent != EventNoEvent {
		return et.lastEvent
	}
	e := <-et.eventChan
	log.Info("got NAT event: ", e)
	return e
}

// Event represents a nat traversal related event
type Event struct {
	Name string
}
