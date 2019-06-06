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

package quality

import (
	"time"
)

const appName = "myst"
const startupEventName = "startup"
const natMappingEventName = "nat_mapping"

// Transport allows sending events
type Transport interface {
	SendEvent(Event) error
}

// NewSender creates metrics sender with appropriate transport
func NewSender(transport Transport, appVersion string) *Sender {
	return &Sender{
		Transport:  transport,
		AppVersion: appVersion,
	}
}

// Sender builds events and sends them using given transport
type Sender struct {
	Transport  Transport
	AppVersion string
}

// Event contains data about event, which is sent using transport
type Event struct {
	Application appInfo     `json:"application"`
	EventName   string      `json:"eventName"`
	CreatedAt   int64       `json:"createdAt"`
	Context     interface{} `json:"context"`
}

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type natMappingContext struct {
	Stage        string              `json:"stage"`
	Successful   bool                `json:"successful"`
	ErrorMessage *string             `json:"error_message"`
	Gateways     []map[string]string `json:"gateways,omitempty"`
}

// SendStartupEvent sends startup event
func (sender *Sender) SendStartupEvent() error {
	return sender.sendEvent(startupEventName, nil)
}

// SendNATMappingSuccessEvent sends event about successful NAT mapping
func (sender *Sender) SendNATMappingSuccessEvent(stage string, gateways []map[string]string) error {
	context := natMappingContext{Stage: stage, Successful: true, Gateways: gateways}
	return sender.sendEvent(natMappingEventName, context)
}

// SendNATMappingFailEvent sends event about failed NAT mapping
func (sender *Sender) SendNATMappingFailEvent(stage string, gateways []map[string]string, err error) error {
	errorMessage := err.Error()
	context := natMappingContext{Stage: stage, Successful: false, ErrorMessage: &errorMessage, Gateways: gateways}
	return sender.sendEvent(natMappingEventName, context)
}

func (sender *Sender) sendEvent(eventName string, context interface{}) error {
	app := appInfo{Name: appName, Version: sender.AppVersion}
	event := Event{Application: app, EventName: eventName, CreatedAt: time.Now().Unix(), Context: context}
	return sender.Transport.SendEvent(event)
}
