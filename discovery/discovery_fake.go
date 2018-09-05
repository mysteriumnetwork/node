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

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysterium/node/identity/registry"
	"github.com/mysterium/node/server"
)

// NewFakeDiscrovery creates fake discovery structure
func NewFakeDiscrovery() *Discovery {
	return &Discovery{
		proposalStatusChan:          make(chan ProposalStatus),
		proposalAnnouncementStopped: &sync.WaitGroup{},
		ownIdentity:                 common.Address{},
		registrationDataProvider:    &registry.FakeRegistrationDataProvider{},
		mysteriumClient:             server.NewClientFake(),
	}
}
