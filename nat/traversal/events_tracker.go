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

const EventTopic = "Traversal"

var EventNoEvent = Event{Name: ""}
var EventSuccess = Event{Name: "success"}
var EventFailure = Event{Name: "failure"}

type EventsTracker struct {
	lastEvent Event
	eventChan chan Event
}

func NewEventsTracker() *EventsTracker {
	return &EventsTracker{eventChan: make(chan Event, 1)}
}

func (et *EventsTracker) ConsumeNATEvent(event Event) {
	log.Info("got NAT event: ", event)
	et.lastEvent = event
	select {
	case et.eventChan <- event:
	case <-time.After(300 * time.Millisecond):
	}
}

func (et *EventsTracker) LastEvent() Event {
	log.Info("getting last NAT event: ", et.lastEvent)
	return et.lastEvent
}

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
