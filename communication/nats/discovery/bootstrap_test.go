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
	"encoding/json"
	"testing"

	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/stretchr/testify/assert"
)

func init() {
	// TODO Use transport mock in tests instead of real openvpn
	openvpn.Bootstrap()
	Bootstrap()
}

func TestServiceProposalUnserializeNatsContact(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {},
		"payment_method_type": "PER_TIME",
		"payment_method": {},
		"provider_contacts": [
			{
				"type": "nats/v1",
				"definition": {
					"topic": "test-topic"
				}
			}
		]
	}`)

	var actual dto_discovery.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Len(t, actual.ProviderContacts, 1)
	assert.Exactly(
		t,
		dto_discovery.Contact{
			Type: TypeContactNATSV1,
			Definition: ContactNATSV1{
				Topic: "test-topic",
			},
		},
		actual.ProviderContacts[0],
	)
}
