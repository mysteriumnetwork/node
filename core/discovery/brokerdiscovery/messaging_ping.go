/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package brokerdiscovery

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/market"
)

// pingMessage structure represents message that the Provider sends about healthy Proposal
type pingMessage struct {
	Proposal market.ServiceProposal `json:"proposal"`
}

const pingEndpoint = communication.MessageEndpoint("proposal-ping")

// pingProducer
type pingProducer struct {
	message *pingMessage
}

// GetMessageEndpoint returns endpoint where to send messages
func (p *pingProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return pingEndpoint
}

// Produce creates message which will be serialized to endpoint
func (p *pingProducer) Produce() (requestPtr interface{}) {
	return p.message
}

// pingConsumer
type pingConsumer struct {
	Callback func(pingMessage) error
}

// GetMessageEndpoint returns endpoint where to receive messages
func (c *pingConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return pingEndpoint
}

// NewMessage creates struct where message from endpoint will be serialized
func (c *pingConsumer) NewMessage() (messagePtr interface{}) {
	return &pingMessage{}
}

// Consume handles messages from endpoint
func (c *pingConsumer) Consume(messagePtr interface{}) error {
	return c.Callback(*messagePtr.(*pingMessage))
}
