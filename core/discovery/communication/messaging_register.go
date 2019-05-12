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

// registerMessage structure represents message that the Provider sends about newly announced Proposal
type registerMessage struct {
	Proposal market.ServiceProposal `json:"proposal"`
}

const registerEndpoint = communication.MessageEndpoint("proposal-register")

// Dialog boilerplate below, please ignore

// registerConsumer
type registerConsumer struct {
	queue chan registerMessage
}

// GetMessageEndpoint returns endpoint where to receive messages
func (pmc *registerConsumer) GetMessageEndpoint() communication.MessageEndpoint {
	return registerEndpoint
}

// NewMessage creates struct where message from endpoint will be serialized
func (pmc *registerConsumer) NewMessage() (messagePtr interface{}) {
	return &registerMessage{}
}

// Consume handles messages from endpoint
func (pmc *registerConsumer) Consume(messagePtr interface{}) error {
	pmc.queue <- messagePtr.(registerMessage)
	return nil
}

// registerProducer
type registerProducer struct {
	message *registerMessage
}

// GetMessageEndpoint returns endpoint where to send messages
func (pmp *registerProducer) GetMessageEndpoint() communication.MessageEndpoint {
	return registerEndpoint
}

// Produce creates message which will be serialized to endpoint
func (pmp *registerProducer) Produce() (requestPtr interface{}) {
	return pmp.message
}
