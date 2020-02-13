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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() market.Location {
	return market.Location{ASN: 123, Country: "Lithuania", City: "Vilnius"}
}

const upperTimePriceBound uint64 = 50000
const lowerTimePriceBound uint64 = 0
const upperGBPriceBound uint64 = 7000000
const lowerGBPriceBound uint64 = 0

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
	repository := &mockProposalRepository{
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
	setPricingBounds(query)
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(repository, &mockQualityProvider{}).List
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

	upperTime := upperTimePriceBound
	lowerTime := lowerTimePriceBound
	upperGB := upperGBPriceBound
	lowerGB := lowerGBPriceBound

	assert.EqualValues(t, &proposal.Filter{
		ProviderID:          "0xProviderId",
		UpperTimePriceBound: &upperTime,
		LowerTimePriceBound: &lowerTime,
		UpperGBPriceBound:   &upperGB,
		LowerGBPriceBound:   &lowerGB,
	}, repository.recordedFilter)
}

func TestProposalsEndpointAcceptsAccessPolicyParams(t *testing.T) {
	repository := &mockProposalRepository{
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
	handlerFunc := NewProposalsEndpoint(repository, &mockQualityProvider{}).List
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
		&proposal.Filter{
			AccessPolicyID:     "accessPolicyId",
			AccessPolicySource: "accessPolicySource",
		},
		repository.recordedFilter,
	)
}

func TestProposalsEndpointList(t *testing.T) {
	repository := &mockProposalRepository{
		proposals: serviceProposals,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(repository, &mockQualityProvider{}).List
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
	repository := &mockProposalRepository{
		proposals: serviceProposals,
	}
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant?fetchConnectCounts=true",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()

	handlerFunc := NewProposalsEndpoint(repository, &mockQualityProvider{}).List
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
					}
				}
			]
		}`,
		resp.Body.String(),
	)
}

type mockQualityProvider struct{}

func (m *mockQualityProvider) ProposalsMetrics() []quality.ConnectMetric {
	p1 := serviceProposals[0]
	return []quality.ConnectMetric{
		{
			ProposalID: quality.ProposalID{
				ProviderID:  p1.ProviderID,
				ServiceType: p1.ServiceType,
			},
			ConnectCount: quality.ConnectCount{
				Success: 5,
				Fail:    3,
				Timeout: 2,
			},
		},
	}
}

type mockProposalRepository struct {
	proposals      []market.ServiceProposal
	recordedFilter *proposal.Filter
}

func (m *mockProposalRepository) Proposal(id market.ProposalID) (*market.ServiceProposal, error) {
	if len(m.proposals) == 0 {
		return nil, nil
	}
	return &m.proposals[0], nil
}

func (m *mockProposalRepository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	m.recordedFilter = filter
	return m.proposals, nil
}

func setPricingBounds(v url.Values) {
	v.Add("upperTimePriceBound", fmt.Sprintf("%v", upperTimePriceBound))
	v.Add("lowerTimePriceBound", fmt.Sprintf("%v", lowerTimePriceBound))
	v.Add("upperGBPriceBound", fmt.Sprintf("%v", upperGBPriceBound))
	v.Add("lowerGBPriceBound", fmt.Sprintf("%v", lowerGBPriceBound))
}
