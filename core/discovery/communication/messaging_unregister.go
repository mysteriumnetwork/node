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

package communication

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/market"
)

// unregisterMessage structure represents message that the Provider sends about de-announced Proposal
type unregisterMessage struct {
	Proposal market.ServiceProposal `json:"proposal"`
}

const unregisterEndpoint = communication.MessageEndpoint("proposal-unregister")

// Dialog boilerplate below, please ignore

// unregisterConsumer
type unregisterConsumer struct {
	queue chan unregisterMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *unregisterConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return unregisterEndpoint
}

// NewMessage creates struct where message from endpoint will be serialized
func (pmc *unregisterConsumer) NewMessage() (messagePtr interface{}) {
	return &unregisterMessage{}
}

// Consume handles messages from endpoint
func (pmc *unregisterConsumer) Consume(messagePtr interface{}) error {
	pmc.queue <- messagePtr.(unregisterMessage)
	return nil
}

// unregisterProducer
type unregisterProducer struct {
	message *unregisterMessage
}

// GetMessageEndpoint returns endpoint where to send messages
func (pmp *unregisterProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return unregisterEndpoint
}

// Produce creates message which will be serialized to endpoint
func (pmp *unregisterProducer) Produce() (requestPtr interface{}) {
	return pmp.message
}
