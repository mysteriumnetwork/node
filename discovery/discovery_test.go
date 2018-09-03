/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/identity/registry"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

func TestRegisterIdentitySuccessful(t *testing.T) {
	d := &Discovery{}
	d.proposalStatusChan = make(chan ProposalStatus)
	d.identityRegistry = &registry.FakeRegister{RegistrationEventExists: true}
	d.ownIdentity = common.Address{}

	d.registerIdentity()

	event := <-d.proposalStatusChan
	assert.Equal(t, event, RegisterProposal)
}

func TestRegisterIdentityCancelled(t *testing.T) {
	d := &Discovery{}
	d.proposalStatusChan = make(chan ProposalStatus)
	d.identityRegistry = &registry.FakeRegister{RegistrationEventExists: false}
	d.ownIdentity = common.Address{}

	d.registerIdentity()
	d.unsubscribe()

	event := <-d.proposalStatusChan
	assert.Equal(t, event, IdentityRegisterFailed)
}

func TestUnregisterProposalSuccessful(t *testing.T) {
	d := &Discovery{}
	fakeMystClient := server.NewClientFake()
	d.proposalStatusChan = make(chan ProposalStatus)
	d.mysteriumClient = fakeMystClient
	d.proposal = dto_discovery.ServiceProposal{}
	d.signer = &identity.SignerFake{}

	d.unregisterProposal()

	event := <-d.proposalStatusChan
	assert.Equal(t, event, ProposalUnregistered)
}
