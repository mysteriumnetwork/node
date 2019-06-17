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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() market.Location {
	return market.Location{ASN: 123, Country: "Lithuania", City: "Vilnius"}
}

var serviceProposals = []market.ServiceProposal{
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
	mockProposalProvider := &mockProposalProvider{
		//we assume that underling component does correct filtering
		proposals: []market.ServiceProposal{serviceProposals[0]},
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
	handlerFunc := NewProposalsEndpoint(mockProposalProvider, &mockQualityProvider{}).List
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
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
	assert.Equal(t, &proposalsFilter{providerID: "0xProviderId"}, mockProposalProvider.recordedFilter)
}

func TestProposalsEndpointAcceptsAccessPolicyParams(t *testing.T) {
	mockProposalProvider := &mockProposalProvider{
		proposals: []market.ServiceProposal{serviceProposals[0]},
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("accessPolicyId", "accessPolicyId")
	query.Set("accessPolicySource", "accessPolicySource")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(mockProposalProvider, &mockQualityProvider{}).List
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
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
	assert.Equal(t,
		&proposalsFilter{
			accessPolicyID:     "accessPolicyId",
			accessPolicySource: "accessPolicySource",
		},
		mockProposalProvider.recordedFilter,
	)
}

func TestProposalsEndpointList(t *testing.T) {
	proposalProvider := &mockProposalProvider{
		proposals: serviceProposals,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(proposalProvider, &mockQualityProvider{}).List
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
                            "asn": 123,
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
                            "asn": 123,
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

func TestProposalsEndpointListFetchConnectCounts(t *testing.T) {
	proposalProvider := &mockProposalProvider{
		proposals: serviceProposals,
	}
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant?fetchConnectCounts=true",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(proposalProvider, &mockQualityProvider{}).List
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
							"asn": 123,
							"country": "Lithuania",
							"city": "Vilnius"
						}
					},
					"metrics": {
						"connectCount": {
							"success": 5,
							"fail": 3,
							"timeout": 2
						}
					}
				},
				{
					"id": 1,
					"providerId": "other_provider",
					"serviceType": "testprotocol",
					"serviceDefinition": {
						"locationOriginate": {
							"asn": 123,
							"country": "Lithuania",
							"city": "Vilnius"
						}
					},
					"metrics": {}
				}
			]
		}`,
		resp.Body.String(),
	)
}

type mockQualityProvider struct{}

// ProposalsMetrics returns a list of proposals connection metrics
func (m *mockQualityProvider) ProposalsMetrics() []json.RawMessage {
	for _, proposal := range serviceProposals {
		return []json.RawMessage{json.RawMessage(`{
			"proposalID": {
				"providerID": "` + proposal.ProviderID + `",
				"serviceType": "` + proposal.ServiceType + `"
			},
			"connectCount": {
				"success": 5,
				"fail": 3,
				"timeout": 2
			}
		}`)}
	}
	return nil
}

type mockProposalProvider struct {
	recordedFilter discovery.ProposalFilter
	proposals      []market.ServiceProposal
}

func (mpp *mockProposalProvider) GetProposal(id market.ProposalID) (*market.ServiceProposal, error) {
	if len(mpp.proposals) == 0 {
		return nil, nil
	}
	return &mpp.proposals[0], nil
}

func (mpp *mockProposalProvider) FindProposals(filter discovery.ProposalFilter) ([]market.ServiceProposal, error) {
	mpp.recordedFilter = filter
	return mpp.proposals, nil
}

var _ ProposalFinder = &mockProposalProvider{}
