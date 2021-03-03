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
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
)

type TestServiceDefinition struct{}

func (service TestServiceDefinition) GetLocation() market.Location {
	return market.Location{ASN: 123, Country: "Lithuania", City: "Vilnius"}
}

var (
	upperTimePriceBound = big.NewInt(50000)
	lowerTimePriceBound = big.NewInt(0)
	upperGBPriceBound   = big.NewInt(7000000)
	lowerGBPriceBound   = big.NewInt(0)
)

var serviceProposals = []market.ServiceProposal{
	{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
		PaymentMethodType: mocks.DefaultPaymentMethodType,
		PaymentMethod:     mocks.DefaultPaymentMethod(),
	},
	{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "other_provider",
		PaymentMethodType: mocks.DefaultPaymentMethodType,
		PaymentMethod:     mocks.DefaultPaymentMethod(),
	},
}

func TestProposalsEndpointListByNodeId(t *testing.T) {
	repository := &mockProposalRepository{
		// we assume that underling component does correct filtering
		proposals: []market.ServiceProposal{serviceProposals[0]},
	}

	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("provider_id", "0xProviderId")
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
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "service_definition": {
                        "location_originate": {
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    },
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount": 50000,
							"currency": "MYST"
						},
						"rate": {
							"per_seconds": 60,
							"per_bytes": 7669584
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
		UpperTimePriceBound: upperTime,
		LowerTimePriceBound: lowerTime,
		UpperGBPriceBound:   upperGB,
		LowerGBPriceBound:   lowerGB,
		ExcludeUnsupported:  true,
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
	query.Set("access_policy_id", "accessPolicyId")
	query.Set("access_policy_source", "accessPolicySource")
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
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "service_definition": {
                        "location_originate": {
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    },
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
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
			ExcludeUnsupported: true,
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
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "service_definition": {
                        "location_originate": {
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    },
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					}
                },
                {
                    "id": 1,
                    "provider_id": "other_provider",
                    "service_type": "testprotocol",
                    "service_definition": {
                        "location_originate": {
                            "asn": 123,
                            "country": "Lithuania",
                            "city": "Vilnius"
                        }
                    },
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
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
		"/irrelevant?fetch_quality=true",
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
					"provider_id": "0xProviderId",
					"service_type": "testprotocol",
					"service_definition": {
						"location_originate": {
							"asn": 123,
							"country": "Lithuania",
							"city": "Vilnius"
						}
					},
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					},
					"metrics": {
						"connect_count": {
							"success": 5,
							"fail": 3,
							"timeout": 2
						},
						"monitoring_failed": false
					}
				},
				{
					"id": 1,
					"provider_id": "other_provider",
					"service_type": "testprotocol",
					"service_definition": {
						"location_originate": {
							"asn": 123,
							"country": "Lithuania",
							"city": "Vilnius"
						}
					},
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					}
				}
			]
		}`,
		resp.Body.String(),
	)
}

type mockQualityProvider struct{}

func (m *mockQualityProvider) ProposalsQuality() []quality.ProposalQuality {
	p1 := serviceProposals[0]
	return []quality.ProposalQuality{
		{
			ProposalID: quality.ProposalID{
				ProviderID:  p1.ProviderID,
				ServiceType: p1.ServiceType,
			},
			Quality: 2,
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
	v.Add("upper_time_price_bound", fmt.Sprintf("%v", upperTimePriceBound))
	v.Add("lower_time_price_bound", fmt.Sprintf("%v", lowerTimePriceBound))
	v.Add("upper_gb_price_bound", fmt.Sprintf("%v", upperGBPriceBound))
	v.Add("lower_gb_price_bound", fmt.Sprintf("%v", lowerGBPriceBound))
}
