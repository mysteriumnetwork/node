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

	log "github.com/cihub/seelog"
)

// RoleProvider indicates that event was sent from service provider
const RoleProvider = "provider"

const appName = "myst"
const startupEventName = "startup"

// Sender builds events and sends them using given transport
type Sender struct {
	Transport Transport
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

type startupContext struct {
	Role string `json:"role"`
}

// SendStartupEvent sends startup event
func (sender *Sender) SendStartupEvent(role, version string) {
	appInfo := applicationInfo{Name: appName, Version: version}
	event := event{Application: appInfo, EventName: startupEventName, Context: startupContext{Role: role}, CreatedAt: time.Now().Unix()}

	sender.sendEvent(event)
}

func (sender *Sender) sendEvent(event event) {
	go func() {
		err := sender.Transport.sendEvent(event)

		if err != nil {
			log.Warnf("sending event failed: %v", err)
		}
	}()
}
