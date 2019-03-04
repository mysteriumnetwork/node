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

package metrics

import (
	"time"
)

const appName = "myst"
const startupEventName = "startup"
const natMappingSuccessEventName = "nat_mapping_success"
const natMappingFailEventName = "nat_mapping_fail"

// Sender builds events and sends them using given transport
type Sender struct {
	Transport          Transport
	ApplicationVersion string
}

// Transport allows sending events
type Transport interface {
	sendEvent(event) error
}

type event struct {
	Application applicationInfo `json:"application"`
	EventName   string          `json:"eventName"`
	CreatedAt   int64           `json:"createdAt"`
	Context     interface{}     `json:"context"`
}

type applicationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type natMappingFailContext struct {
	error string
}

// SendStartupEvent sends startup event
func (sender *Sender) SendStartupEvent() error {
	return sender.sendEvent(startupEventName, nil)
}

// SendNATMappingSuccessEvent sends event about successful NAT mapping
func (sender *Sender) SendNATMappingSuccessEvent() error {
	return sender.sendEvent(natMappingSuccessEventName, nil)
}

// SendNATMappingFailEvent sends event about failed NAT mapping
func (sender *Sender) SendNATMappingFailEvent(err error) error {
	context := natMappingFailContext{error: err.Error()}
	return sender.sendEvent(natMappingFailEventName, context)
}

func (sender *Sender) sendEvent(eventName string, context interface{}) error {
	appInfo := applicationInfo{Name: appName, Version: sender.ApplicationVersion}
	event := event{Application: appInfo, EventName: eventName, CreatedAt: time.Now().Unix(), Context: context}
	return sender.Transport.sendEvent(event)
}
