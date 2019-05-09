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

package discovery

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

type registryBroker struct {
}

// NewRegistryBroker create instance if Broker registry
func NewRegistryBroker() *registryBroker {
	return &registryBroker{}
}

// RegisterProposal registers service proposal to discovery service
func (registry *registryBroker) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	// TODO implement here
	return nil
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (registry *registryBroker) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	// TODO implement here
	return nil
}

// PingProposal pings service proposal as being alive
func (registry *registryBroker) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	// TODO implement here
	return nil
}
