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

package brokerdiscovery

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

type registryBroker struct {
	sender communication.Sender
}

// NewRegistry create an instance of Broker registryBroker
func NewRegistry(connection nats.Connection) *registryBroker {
	return &registryBroker{
		sender: nats.NewSender(connection, communication.NewCodecJSON(), "*"),
	}
}

// RegisterProposal registers service proposal to discovery service
func (rb *registryBroker) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &registerMessage{Proposal: proposal}
	return rb.sender.Send(&registerProducer{message: message})
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (rb *registryBroker) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &unregisterMessage{Proposal: proposal}
	return rb.sender.Send(&unregisterProducer{message: message})
}

// PingProposal pings service proposal as being alive
func (rb *registryBroker) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &pingMessage{Proposal: proposal}
	return rb.sender.Send(&pingProducer{message: message})
}
