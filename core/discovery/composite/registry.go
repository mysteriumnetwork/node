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

package composite

import (
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
)

type registryComposite struct {
	registries []discovery.ProposalRegistry
}

// NewRegistry creates an instance of composite registry
func NewRegistry(registries ...discovery.ProposalRegistry) *registryComposite {
	return &registryComposite{registries: registries}
}

// AddRegistry adds registry to set of registries
func (rc *registryComposite) AddRegistry(registry discovery.ProposalRegistry) {
	rc.registries = append(rc.registries, registry)
}

// RegisterProposal registers service proposal to discovery service
func (rc *registryComposite) RegisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	for _, registry := range rc.registries {
		if err := registry.RegisterProposal(proposal, signer); err != nil {
			return errors.Wrapf(err, "failed to register proposal: %v", proposal)

		}
	}

	return nil
}

// UnregisterProposal unregisters a service proposal when client disconnects
func (rc *registryComposite) UnregisterProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	for _, registry := range rc.registries {
		if err := registry.UnregisterProposal(proposal, signer); err != nil {
			return errors.Wrapf(err, "failed to unregister proposal: %v", proposal)
		}
	}

	return nil
}

// PingProposal pings service proposal as being alive
func (rc *registryComposite) PingProposal(proposal market.ServiceProposal, signer identity.Signer) error {
	for _, registry := range rc.registries {
		if err := registry.PingProposal(proposal, signer); err != nil {
			return errors.Wrapf(err, "failed to ping proposal: %v", proposal)
		}
	}

	return nil
}
