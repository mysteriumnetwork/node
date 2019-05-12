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
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// NewSender creates sender communication sender thru NATS
func NewSender(connection nats.Connection) communication.Sender {
	return nats.NewSender(
		connection,
		communication.NewCodecJSON(),
		"*",
	)
}

type registry struct {
	sender communication.Sender
}

// NewRegistry create instance if Broker registry
func NewRegistry(sender communication.Sender) *registry {
	return &registry{
		sender: sender,
	}
}

// RegisterProposal registers service proposal to discovery service
func (registry *registry) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &registerMessage{Proposal: proposal}
	return registry.sender.Send(&registerProducer{message: message})
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (registry *registry) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &unregisterMessage{Proposal: proposal}
	return registry.sender.Send(&unregisterProducer{message: message})
}

// PingProposal pings service proposal as being alive
func (registry *registry) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	message := &registerMessage{Proposal: proposal}
	return registry.sender.Send(&registerProducer{message: message})
}
