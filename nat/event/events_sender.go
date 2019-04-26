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
	log "github.com/cihub/seelog"
)

const eventsSenderLogPrefix = "[traversal-events-sender] "

// EventsSender allows subscribing to NAT events and sends them to server
type EventsSender struct {
	metricsSender metricsSender
	ipResolver    ipResolver
	lastIp        string
	lastEvent     *Event
}

type metricsSender interface {
	SendNATMappingSuccessEvent(stage string) error
	SendNATMappingFailEvent(stage string, err error) error
}

type ipResolver func() (string, error)

// NewEventsSender returns a new instance of events sender
func NewEventsSender(metricsSender metricsSender, ipResolver ipResolver) *EventsSender {
	return &EventsSender{metricsSender: metricsSender, ipResolver: ipResolver, lastIp: ""}
}

// ConsumeNATEvent sends received event to server
func (es *EventsSender) ConsumeNATEvent(event Event) {
	publicIP, err := es.ipResolver()
	if err != nil {
		log.Warnf(eventsSenderLogPrefix, "resolving public ip failed: ", err)
		return
	}
	if publicIP == es.lastIp && es.matchesLastEvent(event) {
		return
	}

	err = es.sendNATEvent(event)
	if err != nil {
		log.Warnf(eventsSenderLogPrefix, "sending event failed: ", err)
	}

	es.lastIp = publicIP
	es.lastEvent = &event
}

func (es *EventsSender) sendNATEvent(event Event) error {
	if event.Successful {
		return es.metricsSender.SendNATMappingSuccessEvent(event.Stage)
	}

	return es.metricsSender.SendNATMappingFailEvent(event.Stage, event.Error)
}

func (es *EventsSender) matchesLastEvent(event Event) bool {
	if es.lastEvent == nil {
		return false
	}

	return event.Successful == es.lastEvent.Successful && event.Stage == es.lastEvent.Stage
}
