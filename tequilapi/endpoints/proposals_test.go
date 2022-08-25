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
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/nat"
)

var TestLocation = market.Location{ASN: 123, Country: "Lithuania", City: "Vilnius"}

var (
	priceHourMax = big.NewInt(50000)
	priceGiBMax  = big.NewInt(7000000)
	mockQuality  = mocks.Quality()
)

var serviceProposals = []proposal.PricedServiceProposal{
	{
		ServiceProposal: market.NewProposal("0xProviderId", "testprotocol", market.NewProposalOpts{
			Location: &TestLocation,
			Quality:  &mockQuality,
		}),
		Price: market.Price{
			PricePerHour: big.NewInt(500_000_000_000_000_000),
			PricePerGiB:  big.NewInt(1_000_000_000_000_000_000),
		},
	},
	{
		ServiceProposal: market.NewProposal("other_provider", "testprotocol", market.NewProposalOpts{
			Location: &TestLocation,
			Quality:  &mockQuality,
		}),
		Price: market.Price{
			PricePerHour: big.NewInt(500_000_000_000_000_000),
			PricePerGiB:  big.NewInt(1_000_000_000_000_000_000),
		},
	},
}

type mockNATProber struct {
	returnRes nat.NATType
	returnErr error
}

func (m *mockNATProber) Probe(_ context.Context) (nat.NATType, error) {
	return m.returnRes, m.returnErr
}

var mockedNATProber = &mockNATProber{"none", nil}

type mockResolver struct{}

func (r *mockResolver) DetectLocation() (locationstate.Location, error) {
	return locationstate.Location{}, nil
}

func (r *mockResolver) DetectProxyLocation(_ int) (locationstate.Location, error) {
	return r.DetectLocation()
}

type mockPricer struct {
	priceToReturn market.Price
}

func (mpip *mockPricer) GetCurrentPrice(nodeType string, country string, serviceType string) (market.Price, error) {
	return mpip.priceToReturn, nil
}

func TestProposalsEndpointListByNodeId(t *testing.T) {
	repository := &mockProposalRepository{
		// we assume that underling component does correct filtering
		proposals: []proposal.PricedServiceProposal{serviceProposals[0]},
	}

	path := "/proposals"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("provider_id", "0xProviderId")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	endpoint := NewProposalsEndpoint(repository, nil, nil, &mockFilterPresetRepository{}, mockedNATProber)
	g := gin.Default()
	g.GET(path, endpoint.List)
	g.ServeHTTP(resp, req)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 2,
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
                      "bandwidth": 10,
                      "uptime": 20
                    },
					"price": {
					  "currency": "MYST",
					  "per_gib": 1000000000000000000,
					  "per_gib_tokens": {
						"ether": "1",
						"human": "1",
						"wei": "1000000000000000000"
					  },
					  "per_hour": 500000000000000000,
					  "per_hour_tokens": {
						"ether": "0.5",
						"human": "0.5",
						"wei": "500000000000000000"
					  }
					}
                }
            ]
        }`,
		resp.Body.String(),
	)

	assert.EqualValues(t, &proposal.Filter{
		ProviderID:         "0xProviderId",
		ExcludeUnsupported: true,
		CompatibilityMin:   2,
	}, repository.recordedFilter)
}

func TestProposalsEndpointAcceptsAccessPolicyParams(t *testing.T) {
	repository := &mockProposalRepository{
		proposals: []proposal.PricedServiceProposal{serviceProposals[0]},
	}
	path := "/proposals"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	query := req.URL.Query()
	query.Set("access_policy", "accessPolicy")
	query.Set("access_policy_source", "accessPolicySource")
	req.URL.RawQuery = query.Encode()

	resp := httptest.NewRecorder()
	endpoint := NewProposalsEndpoint(repository, nil, nil, &mockFilterPresetRepository{}, mockedNATProber)

	g := gin.Default()
	g.GET(path, endpoint.List)
	g.ServeHTTP(resp, req)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 2,
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
                      "bandwidth": 10,
                      "uptime": 20
                    },
					"price": {
					  "currency": "MYST",
					  "per_gib": 1000000000000000000,
					  "per_gib_tokens": {
						"ether": "1",
						"human": "1",
						"wei": "1000000000000000000"
					  },
					  "per_hour": 500000000000000000,
					  "per_hour_tokens": {
						"ether": "0.5",
						"human": "0.5",
						"wei": "500000000000000000"
					  }
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
			CompatibilityMin:   2,
		},
		repository.recordedFilter,
	)
}

func TestCurrentPrices(t *testing.T) {
	// given
	repository := &mockProposalRepository{
		proposals: serviceProposals,
	}
	presetRepository := &mockFilterPresetRepository{
		presets: proposal.FilterPresets{Entries: []proposal.FilterPreset{
			{
				ID:     0,
				Name:   "",
				IPType: "",
			},
		}},
	}
	endpoint := NewProposalsEndpoint(repository, &mockPricer{
		priceToReturn: market.Price{
			PricePerHour: big.NewInt(123_000_000_000_000_000),
			PricePerGiB:  big.NewInt(456_000_000_000_000_000),
		},
	}, &mockResolver{}, presetRepository, mockedNATProber)

	path := "/prices/current"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	g := gin.Default()
	g.GET(path, endpoint.CurrentPrice)
	resp := httptest.NewRecorder()
	g.ServeHTTP(resp, req)

	assert.JSONEq(
		t,
		`
				{
				  "service_type": "wireguard",
				  "price_per_hour": 123000000000000000,
				  "price_per_hour_tokens": {
					"wei": "123000000000000000",
					"ether": "0.123",
					"human": "0.123"
				  },
				  "price_per_gib": 456000000000000000,
				  "price_per_gib_tokens": {
					"wei": "456000000000000000",
					"ether": "0.456",
					"human": "0.456"
				  }
				}
				`,
		resp.Body.String(),
	)
}

func TestProposalsEndpointFilterByPresetID(t *testing.T) {
	repository := &mockProposalRepository{
		proposals: serviceProposals,
	}

	path := "/proposals"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	presetRepository := &mockFilterPresetRepository{
		presets: proposal.FilterPresets{Entries: []proposal.FilterPreset{
			{
				ID:     0,
				Name:   "",
				IPType: "",
			},
		}},
	}
	endpoint := NewProposalsEndpoint(repository, nil, nil, presetRepository, mockedNATProber)
	g := gin.Default()
	g.GET(path, endpoint.List)
	g.ServeHTTP(resp, req)

	assert.JSONEq(
		t,
		`{
            "proposals": [
                {
                    "format": "service-proposal/v3",
                    "compatibility": 2,
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
                      "bandwidth": 10,
                      "uptime": 20
                    },
					"price": {
					  "currency": "MYST",
					  "per_gib": 1000000000000000000,
					  "per_gib_tokens": {
						"ether": "1",
						"human": "1",
						"wei": "1000000000000000000"
					  },
					  "per_hour": 500000000000000000,
					  "per_hour_tokens": {
						"ether": "0.5",
						"human": "0.5",
						"wei": "500000000000000000"
					  }
					}
                },
                {
                    "format": "service-proposal/v3",
                    "compatibility": 2,
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
                      "bandwidth": 10,
                      "uptime": 20
                    },
					"price": {
					  "currency": "MYST",
					  "per_gib": 1000000000000000000,
					  "per_gib_tokens": {
						"ether": "1",
						"human": "1",
						"wei": "1000000000000000000"
					  },
					  "per_hour": 500000000000000000,
					  "per_hour_tokens": {
						"ether": "0.5",
						"human": "0.5",
						"wei": "500000000000000000"
					  }
					}
                }
            ]
        }`,
		resp.Body.String(),
	)
}

type mockProposalRepository struct {
	proposals      []proposal.PricedServiceProposal
	recordedFilter *proposal.Filter
	priceToAdd     market.Price
}

func (m *mockProposalRepository) Proposal(_ market.ProposalID) (*proposal.PricedServiceProposal, error) {
	if len(m.proposals) == 0 {
		return nil, nil
	}
	return &m.proposals[0], nil
}

func (m *mockProposalRepository) Proposals(filter *proposal.Filter) ([]proposal.PricedServiceProposal, error) {
	m.recordedFilter = filter
	return m.proposals, nil
}

func (m *mockProposalRepository) Countries(filter *proposal.Filter) (map[string]int, error) {
	m.recordedFilter = filter
	return nil, nil
}

func (m *mockProposalRepository) EnrichProposalWithPrice(in market.ServiceProposal) (proposal.PricedServiceProposal, error) {
	return proposal.PricedServiceProposal{
		Price:           m.priceToAdd,
		ServiceProposal: in,
	}, nil
}

type mockFilterPresetRepository struct {
	presets proposal.FilterPresets
}

func (m *mockFilterPresetRepository) List() (*proposal.FilterPresets, error) {
	return &m.presets, nil
}

func (m *mockFilterPresetRepository) Get(id int) (*proposal.FilterPreset, error) {
	for _, p := range m.presets.Entries {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, errors.New("preset not found")
}

func setPricingBounds(v url.Values) {
	v.Add("price_hour_max", fmt.Sprintf("%v", priceHourMax))
	v.Add("price_gib_max", fmt.Sprintf("%v", priceGiBMax))
}
