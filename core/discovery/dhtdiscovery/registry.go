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

package dhtdiscovery

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

type registryDHT struct {
}

// NewRegistry create an instance of Broker registryBroker.
func NewRegistry() *registryDHT {
	return &registryDHT{}
}

// RegisterProposal registers service proposal to discovery service.
func (rd *registryDHT) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return nil
}

// UnregisterProposal unregisters a service proposal when client disconnects.
func (rd *registryDHT) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return nil
}

// PingProposal pings service proposal as being alive.
func (rd *registryDHT) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return nil
}
