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

const senderLogPrefix = "[traversal-events-sender] "

// Sender allows subscribing to NAT events and sends them to server
type Sender struct {
	metricsSender metricsSender
	ipResolver    ipResolver
	gatewayLoader func() []map[string]string
	lastIp        string
	lastEvent     *Event
}

type metricsSender interface {
	SendNATMappingSuccessEvent(stage string, gateways []map[string]string) error
	SendNATMappingFailEvent(stage string, gateways []map[string]string, err error) error
}

type ipResolver func() (string, error)

// NewSender returns a new instance of events sender
func NewSender(metricsSender metricsSender, ipResolver ipResolver, gatewayLoader func() []map[string]string) *Sender {
	return &Sender{
		metricsSender: metricsSender,
		ipResolver:    ipResolver,
		lastIp:        "",
		gatewayLoader: gatewayLoader,
	}
}

// ConsumeNATEvent sends received event to server
func (es *Sender) ConsumeNATEvent(event Event) {
	publicIP, err := es.ipResolver()
	if err != nil {
		log.Warnf(senderLogPrefix, "resolving public ip failed: ", err)
		return
	}
	if publicIP == es.lastIp && es.matchesLastEvent(event) {
		return
	}

	err = es.sendNATEvent(event)
	if err != nil {
		log.Warnf(senderLogPrefix, "sending event failed: ", err)
	}

	es.lastIp = publicIP
	es.lastEvent = &event
}

func (es *Sender) sendNATEvent(event Event) error {
	if event.Successful {
		return es.metricsSender.SendNATMappingSuccessEvent(event.Stage, es.gatewayLoader())
	}

	return es.metricsSender.SendNATMappingFailEvent(event.Stage, es.gatewayLoader(), event.Error)
}

func (es *Sender) matchesLastEvent(event Event) bool {
	if es.lastEvent == nil {
		return false
	}

	return event.Successful == es.lastEvent.Successful && event.Stage == es.lastEvent.Stage
}
