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
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
)

var TestLocation = market.Location{ASN: 123, Country: "Lithuania", City: "Vilnius"}

var (
	priceHourMax = big.NewInt(50000)
	priceGiBMax  = big.NewInt(7000000)
	mockQuality  = mocks.Quality()
)

var serviceProposals = []market.ServiceProposal{
	market.NewProposal("0xProviderId", "testprotocol", market.NewProposalOpts{
		Location: &TestLocation,
		Quality:  &mockQuality,
	}),
	market.NewProposal("other_provider", "testprotocol", market.NewProposalOpts{
		Location: &TestLocation,
		Quality:  &mockQuality,
	}),
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
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(repository).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 0,
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "location": {
						"asn": 123,
						"country": "Lithuania",
						"city": "Vilnius"
					},
                    "quality": {
                      "quality": 2.0,
                      "latency": 50,
                      "bandwidth": 10
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)

	assert.EqualValues(t, &proposal.Filter{
		ProviderID:         "0xProviderId",
		ExcludeUnsupported: true,
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
	query.Set("access_policy", "accessPolicy")
	query.Set("access_policy_source", "accessPolicySource")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	handlerFunc := NewProposalsEndpoint(repository).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 0,
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "location": {
						"asn": 123,
						"country": "Lithuania",
						"city": "Vilnius"
					},
                    "quality": {
                      "quality": 2.0,
                      "latency": 50,
                      "bandwidth": 10
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
	assert.Equal(t,
		&proposal.Filter{
			AccessPolicy:       "accessPolicy",
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
	handlerFunc := NewProposalsEndpoint(repository).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 0,
                    "provider_id": "0xProviderId",
                    "service_type": "testprotocol",
                    "location": {
						"asn": 123,
						"country": "Lithuania",
						"city": "Vilnius"
					},
                    "quality": {
                      "quality": 2.0,
                      "latency": 50,
                      "bandwidth": 10
                    }
                },
                {
                    "format": "service-proposal/v3",
                    "compatibility": 0,
                    "provider_id": "other_provider",
                    "service_type": "testprotocol",
                    "location": {
						"asn": 123,
						"country": "Lithuania",
						"city": "Vilnius"
					},
                    "quality": {
                      "quality": 2.0,
                      "latency": 50,
                      "bandwidth": 10
                    }
                }
            ]
        }`,
		resp.Body.String(),
	)
}

type mockProposalRepository struct {
	proposals      []market.ServiceProposal
	recordedFilter *proposal.Filter
}

func (m *mockProposalRepository) Proposal(_ market.ProposalID) (*market.ServiceProposal, error) {
	if len(m.proposals) == 0 {
		return nil, nil
	}
	return &m.proposals[0], nil
}

func (m *mockProposalRepository) Proposals(filter *proposal.Filter) ([]market.ServiceProposal, error) {
	m.recordedFilter = filter
	return m.proposals, nil
}
