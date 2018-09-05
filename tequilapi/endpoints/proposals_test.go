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

package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mysteriumnetwork/node/server"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() dto_discovery.Location {
	return dto_discovery.Location{ASN: "LT", Country: "Lithuania", City: "Vilnius"}
}

var proposals = []dto_discovery.ServiceProposal{
	{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
	},
	{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "other_provider",
	},
}

func TestProposalsEndpointListByNodeId(t *testing.T) {
	discoveryAPI := server.NewClientFake()
	for _, proposal := range proposals {
		discoveryAPI.RegisterProposal(proposal, nil)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("providerId", "0xProviderId")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(discoveryAPI).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "id": 1,
                    "providerId": "0xProviderId",
                    "serviceType": "testprotocol",
                    "serviceDefinition": {
                        "locationOriginate": {
                            "asn": "LT",
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
}

func TestProposalsEndpointList(t *testing.T) {
	discoveryAPI := server.NewClientFake()
	for _, proposal := range proposals {
		discoveryAPI.RegisterProposal(proposal, nil)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(discoveryAPI).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "id": 1,
                    "providerId": "0xProviderId",
                    "serviceType": "testprotocol",
                    "serviceDefinition": {
                        "locationOriginate": {
                            "asn": "LT",
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    }
                },
                {
                    "id": 1,
                    "providerId": "other_provider",
                    "serviceType": "testprotocol",
                    "serviceDefinition": {
                        "locationOriginate": {
                            "asn": "LT",
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
}
