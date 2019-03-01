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
const natMappingResultEventName = "nat_mapping_result"

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

type natMappingResultContext struct {
	success bool
}

// SendStartupEvent sends startup event
func (sender *Sender) SendStartupEvent() error {
	return sender.sendEvent(startupEventName, nil)
}

// SendNATMappingResultEvent sends event about NAT mapping result, either successful or unsuccessful
func (sender *Sender) SendNATMappingResultEvent(success bool) error {
	context := natMappingResultContext{success: success}
	return sender.sendEvent(natMappingResultEventName, context)
}

func (sender *Sender) sendEvent(eventName string, context interface{}) error {
	appInfo := applicationInfo{Name: appName, Version: sender.ApplicationVersion}
	event := event{Application: appInfo, EventName: eventName, CreatedAt: time.Now().Unix(), Context: context}
	return sender.Transport.sendEvent(event)
}
