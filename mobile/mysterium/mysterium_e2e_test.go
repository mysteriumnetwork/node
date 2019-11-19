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

package mysterium

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Rewrite to unit test.
func XTestProposals(t *testing.T) {
	node, err := NewNode(os.TempDir(), DefaultLogOptions(), DefaultNetworkOptions())
	assert.NoError(t, err)

	proposalsData, err := node.GetProposals(&GetProposalsRequest{
		ShowOpenvpnProposals:   true,
		ShowWireguardProposals: true,
	})

	assert.NoError(t, err)
	assert.NotEqual(t, `{"proposals":null}`, string(proposalsData))

	var proposalsRes getProposalsResponse
	err = json.Unmarshal(proposalsData, &proposalsRes)
	assert.NoError(t, err)

	proposalData, err := node.GetProposal(&GetProposalRequest{
		ProviderID:  proposalsRes.Proposals[0].ProviderID,
		ServiceType: proposalsRes.Proposals[0].ServiceType,
	})

	assert.NoError(t, err)
	assert.NotEqual(t, `{"proposal":null}`, string(proposalData))

	var proposalRes getProposalResponse
	err = json.Unmarshal(proposalData, &proposalRes)
	assert.NoError(t, err)
	assert.Equal(t, proposalsRes.Proposals[0], proposalRes.Proposal)
}
