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

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn"
	dto_openvpn "github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/stretchr/testify/assert"
)

func init() {
	openvpn.Bootstrap()
}

func Test_ServiceProposal_UnserializeServiceDefinition(t *testing.T) {
	jsonData := []byte(`{
		"service_type": "openvpn",
		"service_definition": {}
	}`)

	var actual market.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	expected := market.ServiceProposal{
		ServiceType:       "openvpn",
		ServiceDefinition: dto_openvpn.ServiceDefinition{},
		PaymentMethod:     market.UnsupportedPaymentMethod{},
		ProviderContacts:  market.ContactList{},
	}
	assert.Equal(t, expected, actual)
}

func Test_ServiceProposal_SerializeServiceDefinition(t *testing.T) {
	sp := market.ServiceProposal{
		ServiceType:       "openvpn",
		ServiceDefinition: dto_openvpn.ServiceDefinition{},
	}

	actualJSON, err := json.Marshal(sp)
	assert.NoError(t, err)

	expectedJSON := `{
	  "id": 0,
	  "format": "",
	  "service_type": "openvpn",
	  "service_definition": {
	    "location": {},
	    "location_originate": {}
	  },
	  "payment_method_type": "",
	  "payment_method": null,
	  "provider_id": "",
	  "provider_contacts": []
	}`
	assert.JSONEq(t, expectedJSON, string(actualJSON))
}

func Test_ServiceProposal_UnserializePerTimePaymentMethod(t *testing.T) {
	jsonData := []byte(`{
		"payment_method_type": "PER_TIME",
		"payment_method": {}
	}`)

	var actual market.ServiceProposal
	err := json.Unmarshal(jsonData, &actual)

	assert.Nil(t, err)
	assert.Exactly(t, dto_openvpn.PaymentRate{}, actual.PaymentMethod)
}
