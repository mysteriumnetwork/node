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

	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/stretchr/testify/assert"
)

var (
	locationLTTelia = location.Location{
		Continent: "EU",
		Country:   "LT",
		City:      "Vilnius",
		ASN:       8764,
		ISP:       "Telia Lietuva, AB",
		NodeType:  "residential",
	}
	protocol = "tcp"
)

func Test_NewServiceProposalWithLocation(t *testing.T) {
	proposal := NewServiceProposalWithLocation(locationLTTelia, protocol)

	assert.Exactly(
		t,
		market.ServiceProposal{
			ServiceType: "openvpn",
			ServiceDefinition: dto.ServiceDefinition{
				Location:          proposal.ServiceDefinition.GetLocation(),
				LocationOriginate: proposal.ServiceDefinition.GetLocation(),
				SessionBandwidth:  83886080,
				Protocol:          "tcp",
			},

			PaymentMethodType: pingpong.DefaultPaymentMethod.GetType(),
			PaymentMethod:     pingpong.DefaultPaymentMethod,
		},
		proposal,
	)
}
