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

package noop

import (
	"testing"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

var (
	providerID = identity.FromAddress("my-identity")
	proposal   = dto.ServiceProposal{
		ProviderID: providerID.Address,
	}
)

func TestPromiseIssuer_Interface(t *testing.T) {
	var _ connection.PromiseIssuer = &PromiseIssuer{}
}

func TestPromiseIssuer_Start(t *testing.T) {
	issuer := &PromiseIssuer{}

	err := issuer.Start(proposal)
	assert.NoError(t, err)
}
