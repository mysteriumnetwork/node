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

package failover

import (
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/utils"
)

type registryFailover struct {
	registryPrimary   discovery.ProposalRegistry
	registrySecondary discovery.ProposalRegistry
}

// NewRegistry create an instance of composite registry
func NewRegistry(registryPrimary discovery.ProposalRegistry, registrySecondary discovery.ProposalRegistry) *registryFailover {
	return &registryFailover{
		registryPrimary:   registryPrimary,
		registrySecondary: registrySecondary,
	}
}

// RegisterProposal registers service proposal to discovery service
func (registry *registryFailover) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	var err utils.ErrorCollection
	err.Add(registry.registryPrimary.RegisterProposal(proposal, signer))
	err.Add(registry.registrySecondary.RegisterProposal(proposal, signer))
	if len(err) == 2 {
		return err.Error()
	}

	return nil
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (registry *registryFailover) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	var err utils.ErrorCollection
	err.Add(registry.registryPrimary.UnregisterProposal(proposal, signer))
	err.Add(registry.registrySecondary.UnregisterProposal(proposal, signer))
	if len(err) == 2 {
		return err.Error()
	}

	return nil
}

// PingProposal pings service proposal as being alive
func (registry *registryFailover) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	var err utils.ErrorCollection
	err.Add(registry.registryPrimary.PingProposal(proposal, signer))
	err.Add(registry.registrySecondary.PingProposal(proposal, signer))
	if len(err) == 2 {
		return err.Error()
	}

	return nil
}
