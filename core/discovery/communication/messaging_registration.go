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

// proposalRegistrationMessage structure represents message that the Provider sends about newly announced Proposal
type proposalRegistrationMessage struct {
	Proposal market.ServiceProposal `json:"proposal"`
}

const registrationEndpoint = communication.MessageEndpoint("proposal-register")

// Dialog boilerplate below, please ignore

// registrationConsumer
type registrationConsumer struct {
	queue chan proposalRegistrationMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *registrationConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return registrationEndpoint
}

// NewMessage creates struct where message from endpoint will be serialized
func (pmc *registrationConsumer) NewMessage() (messagePtr interface{}) {
	return &proposalRegistrationMessage{}
}

// Consume handles messages from endpoint
func (pmc *registrationConsumer) Consume(messagePtr interface{}) error {
	pmc.queue <- messagePtr.(proposalRegistrationMessage)
	return nil
}

// registrationProducer
type registrationProducer struct {
	message *proposalRegistrationMessage
}

// GetMessageEndpoint returns endpoint where to send messages
func (pmp *registrationProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return registrationEndpoint
}

// Produce creates message which will be serialized to endpoint
func (pmp *registrationProducer) Produce() (requestPtr interface{}) {
	return pmp.message
}
