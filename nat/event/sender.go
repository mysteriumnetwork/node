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
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

// Sender allows subscribing to NAT events and sends them to server
type Sender struct {
	metricsSender metricsSender
	ipResolver    ipResolver
	gatewayLoader func() []map[string]string
	lastIp        string
	lastEvent     *Event
}

type metricsSender interface {
	SendNATMappingSuccessEvent(id, stage string, gateways []map[string]string)
	SendNATMappingFailEvent(id, stage string, gateways []map[string]string, err error)
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

// Subscribe subscribes to relevant events of event bus.
func (es *Sender) Subscribe(bus eventbus.Subscriber) error {
	return bus.Subscribe(AppTopicTraversal, es.consumeNATEvent)
}

// consumeNATEvent sends received event to server
func (es *Sender) consumeNATEvent(event Event) {
	publicIP, err := es.ipResolver()
	if err != nil {
		log.Warn().Err(err).Msg("Resolving public IP failed")
		return
	}

	if publicIP == es.lastIp && es.matchesLastEvent(event) {
		return
	}

	es.sendNATEvent(event)

	es.lastIp = publicIP
	es.lastEvent = &event
}

func (es *Sender) sendNATEvent(event Event) {
	if event.Successful {
		es.metricsSender.SendNATMappingSuccessEvent(event.ID, event.Stage, es.gatewayLoader())
	} else {
		es.metricsSender.SendNATMappingFailEvent(event.ID, event.Stage, es.gatewayLoader(), event.Error)
	}
}

func (es *Sender) matchesLastEvent(event Event) bool {
	if es.lastEvent == nil {
		return false
	}

	return event.Successful == es.lastEvent.Successful && event.Stage == es.lastEvent.Stage
}
