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

package api

import (
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
)

type registryAPI struct {
	mysteriumAPI *mysterium.MysteriumAPI
}

// NewRegistry create instance if API registry
func NewRegistry(mysteriumAPI *mysterium.MysteriumAPI) *registryAPI {
	return &registryAPI{
		mysteriumAPI: mysteriumAPI,
	}
}

// RegisterProposal registers service proposal to discovery service
func (registry *registryAPI) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return registry.mysteriumAPI.RegisterProposal(proposal, signer)
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (registry *registryAPI) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return registry.mysteriumAPI.UnregisterProposal(proposal, signer)
}

// PingProposal pings service proposal as being alive
func (registry *registryAPI) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	return registry.mysteriumAPI.PingProposal(proposal, signer)
}
