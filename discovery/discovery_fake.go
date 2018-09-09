/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"sync"

	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/server"
)

// NewFakeDiscrovery creates fake discovery structure
func NewFakeDiscrovery() *Discovery {
	return &Discovery{
		proposalStatusChan:          make(chan ProposalStatus),
		proposalAnnouncementStopped: &sync.WaitGroup{},
		signerCreate: func(id identity.Identity) identity.Signer {
			return &identity.SignerFake{}
		},
		identityRegistration: &identity_registry.FakeRegistrationDataProvider{},
		mysteriumClient:      server.NewClientFake(),
	}
}
