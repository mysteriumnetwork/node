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

package market

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/stretchr/testify/assert"
)

type mockContact struct{}

func init() {
	RegisterContactUnserializer("mock_contact",
		func(rawMessage *json.RawMessage) (ContactDefinition, error) {
			return mockContact{}, nil
		},
	)
}

func Test_ServiceProposal_Serialize(t *testing.T) {
	config.Current.SetDefault(config.FlagDefaultCurrency.Name, "MYSTT")
	sp := NewProposal("node", "mock_service", NewProposalOpts{
		Quality: &Quality{
			Quality:   2.0,
			Latency:   5,
			Bandwidth: 100,
			Uptime:    20,
		},
		Contacts: ContactList{},
	})

	jsonBytes, err := json.Marshal(sp)
	assert.Nil(t, err)

	expectedJSON := `{
      "compatibility": 2,
	  "format": "service-proposal/v3",
	  "service_type": "mock_service",
	  "provider_id": "node",
      "location": {},
	  "id": 0,
      "quality": {
        "quality": 2.0,
        "latency": 5,
        "bandwidth": 100,
        "uptime": 20
      },
      "contacts": []
	}`
	assert.JSONEq(t, expectedJSON, string(jsonBytes))
}

func Test_ServiceProposal_Unserialize(t *testing.T) {
	RegisterServiceType("mock_service")
	jsonData := []byte(`{
		"id": 1,
		"format": "service-proposal/v3",
		"provider_id": "node",
		"service_type": "mock_service",
		"contacts": [
			{ "type" : "mock_contact" , "definition" : {}}
		]
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	expected := ServiceProposal{
		ID:          1,
		Format:      proposalFormat,
		ServiceType: "mock_service",
		ProviderID:  "node",
		Contacts: ContactList{
			Contact{
				Type:       "mock_contact",
				Definition: mockContact{},
			},
		},
	}
	assert.Equal(t, expected, actual)
	assert.True(t, actual.IsSupported())
}

func Test_ServiceProposal_UnserializeAccessPolicy(t *testing.T) {
	RegisterServiceType("mock_service")
	jsonData := []byte(`{
		"id": 1,
		"format": "service-proposal/v3",
		"service_type": "mock_service",
		"provider_id": "node",
		"contacts": [
			{ "type" : "mock_contact" , "definition" : {}}
		],
		"access_policies": [{
			"id": "verified-traffic",
			"source": "https://mysterium-oracle.mysterium.network/v1/lists/verified-traffic"
		},
		{
			"id": "dvpn-traffic",
			"source": "https://mysterium-oracle.mysterium.network/v1/lists/dvpn-traffic"
		}]
	}`)

	var actual ServiceProposal
	err := json.Unmarshal(jsonData, &actual)
	assert.NoError(t, err)

	accessPolicies := []AccessPolicy{
		{
			ID:     "verified-traffic",
			Source: "https://mysterium-oracle.mysterium.network/v1/lists/verified-traffic",
		},
		{
			ID:     "dvpn-traffic",
			Source: "https://mysterium-oracle.mysterium.network/v1/lists/dvpn-traffic",
		},
	}
	expected := ServiceProposal{
		ID:          1,
		Format:      proposalFormat,
		ServiceType: "mock_service",
		ProviderID:  "node",
		Contacts: ContactList{
			Contact{
				Type:       "mock_contact",
				Definition: mockContact{},
			},
		},
		AccessPolicies: &accessPolicies,
	}
	assert.Equal(t, expected, actual)
	assert.True(t, actual.IsSupported())
}
