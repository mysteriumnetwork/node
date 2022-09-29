/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services"
	"github.com/stretchr/testify/assert"
)

var (
	mockServiceID             = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	mockAccessPolicyServiceID = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c9")
	mockProviderID            = identity.FromAddress("0xproviderid")
	mockServiceType           = "testprotocol"
	mockServiceOptions        = fancyServiceOptions{
		Foo: "bar",
	}
	mockAccessPolicyEndpoint = "https://some.domain/api/v1/lists/"
	mockProposal             = market.NewProposal(mockProviderID.Address, mockServiceType, market.NewProposalOpts{
		Location: &TestLocation,
		Quality:  &mockQuality,
	})
	ap = []market.AccessPolicy{
		{
			ID:     "verified-traffic",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "verified-traffic"),
		},
		{
			ID:     "0x0000000000000001",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "0x0000000000000001"),
		},
		{
			ID:     "dvpn-traffic",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "dvpn-traffic"),
		},
		{
			ID:     "12312312332132",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "12312312332132"),
		},
	}
	serviceTypeWithAccessPolicy  = "mockAccessPolicyService"
	mockProposalWithAccessPolicy = market.NewProposal(mockProviderID.Address, serviceTypeWithAccessPolicy, market.NewProposalOpts{
		Location:       &TestLocation,
		Quality:        &mockQuality,
		AccessPolicies: ap,
	})
	mockServiceRunning                 = service.NewInstance(mockProviderID, mockServiceType, mockServiceOptions, mockProposal, servicestate.Running, nil, nil, nil)
	mockServiceStopped                 = service.NewInstance(mockProviderID, mockServiceType, mockServiceOptions, mockProposal, servicestate.NotRunning, nil, nil, nil)
	mockServiceRunningWithAccessPolicy = service.NewInstance(mockProviderID, serviceTypeWithAccessPolicy, mockServiceOptions, mockProposalWithAccessPolicy, servicestate.Running, nil, nil, nil)
)

type fancyServiceOptions struct {
	Foo string `json:"foo"`
}

type mockServiceManager struct{}

func (sm *mockServiceManager) Start(_ identity.Identity, serviceType string, _ []string, _ service.Options) (service.ID, error) {
	if serviceType == serviceTypeWithAccessPolicy {
		return mockAccessPolicyServiceID, nil
	}
	return mockServiceID, nil
}
func (sm *mockServiceManager) Stop(id service.ID) error { return nil }
func (sm *mockServiceManager) Service(id service.ID) *service.Instance {
	if id == "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		return mockServiceRunning
	}
	if id == mockAccessPolicyServiceID {
		return mockServiceRunningWithAccessPolicy
	}
	return nil
}
func (sm *mockServiceManager) List(includeAll bool) []*service.Instance {
	return []*service.Instance{
		mockServiceStopped,
	}
}
func (sm *mockServiceManager) ListAll() []*service.Instance {
	return []*service.Instance{mockServiceStopped}
}
func (sm *mockServiceManager) Kill() error { return nil }

var fakeOptionsParser = map[string]services.ServiceOptionsParser{
	"testprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	serviceTypeWithAccessPolicy: func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	"errorprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, errors.New("error")
	},
}

type mockTequilaApiClient struct{}

func (c *mockTequilaApiClient) Post(path string, payload interface{}) (*http.Response, error) {
	return nil, nil
}

func Test_AddRoutesForServiceAddsRoutes(t *testing.T) {
	router := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{
		priceToAdd: market.Price{
			PricePerHour: big.NewInt(500_000_000_000_000_000),
			PricePerGiB:  big.NewInt(1_000_000_000_000_000_000),
		},
	}, nil)(router)
	assert.NoError(t, err)
	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet,
			"/services",
			"",
			http.StatusOK,
			`[{
				"options": {"foo": "bar"},
				"provider_id": "0xproviderid",
				"type": "testprotocol",
				"status": "NotRunning"
			}]`,
		},
		{
			http.MethodPost,
			"/services?ignore_user_config=true",
			`{"provider_id": "node1", "type": "testprotocol"}`,
			http.StatusCreated,
			`{
				"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				"provider_id": "0xproviderid",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
		            "format": "service-proposal/v3",
		            "compatibility": 2,
					"provider_id": "0xproviderid",
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
			}`,
		},
		{
			http.MethodGet,
			"/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"",
			http.StatusOK,
			`{
				"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				"provider_id": "0xproviderid",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
		            "format": "service-proposal/v3",
		            "compatibility": 2,
					"provider_id": "0xproviderid",
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
			}`,
		},
		{
			http.MethodDelete, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8?ignore_user_config=true", "",
			http.StatusAccepted, "",
		},
		{
			http.MethodDelete, "/services/00000000-9dad-11d1-80b4-00c04fd43000", "",
			http.StatusNotFound, `{ "error": {"code":"not_found", "message":"Service not found"}, "path":"/services/00000000-9dad-11d1-80b4-00c04fd43000", "status":404 }`,
		},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, test.expectedStatus, resp.Code)
		if test.expectedJSON != "" {
			assert.JSONEq(t, test.expectedJSON, resp.Body.String())
		} else {
			assert.Equal(t, "", resp.Body.String())
		}
	}
}

func Test_ServiceStartInvalidType(t *testing.T) {
	path := "/services"
	req := httptest.NewRequest(
		http.MethodPost,
		path,
		strings.NewReader(`{
			"type": "openvpn",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	apiErr := apierror.Parse(resp.Result())
	assert.Equal(t, "validation_failed", apiErr.Err.Code)
	assert.Contains(t, apiErr.Err.Fields, "type")
	assert.Equal(t, "invalid_value", apiErr.Err.Fields["type"].Code)
}

func Test_ServiceStart_InvalidType(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services",
		strings.NewReader(`{
			"type": "openvpn",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	apiErr := apierror.Parse(resp.Result())
	assert.Equal(t, "validation_failed", apiErr.Err.Code)
	assert.Contains(t, apiErr.Err.Fields, "type")
	assert.Equal(t, "invalid_value", apiErr.Err.Fields["type"].Code)
}

func Test_ServiceStart_InvalidOptions(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services",
		strings.NewReader(`{
			"type": "errorprotocol",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	apiErr := apierror.Parse(resp.Result())
	assert.Equal(t, "validation_failed", apiErr.Err.Code)
	assert.Contains(t, apiErr.Err.Fields, "options")
	assert.Equal(t, "invalid_value", apiErr.Err.Fields["options"].Code)
}

func Test_ServiceStartAlreadyRunning(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services",
		strings.NewReader(`{
			"type": "testprotocol",
			"provider_id": "0xproviderid",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.Equal(t, "err_service_running", apierror.Parse(resp.Result()).Err.Code)
}

func Test_ServiceStatus_NotFoundIsReturnedWhenNotStarted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/services/1", nil)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_ServiceGetReturnsServiceInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8", nil)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(
		&mockServiceManager{},
		fakeOptionsParser,
		&mockProposalRepository{
			priceToAdd: market.Price{
				PricePerHour: big.NewInt(500_000_000_000_000_000),
				PricePerGiB:  big.NewInt(1_000_000_000_000_000_000),
			},
		},
		nil,
	)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"provider_id": "0xproviderid",
			"type": "testprotocol",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
				"format": "service-proposal/v3",
				"compatibility": 2,
				"provider_id": "0xproviderid",
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
		}`,
		resp.Body.String(),
	)
}
func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "parse_failed", apierror.Parse(resp.Result()).Err.Code)
}

func Test_ServiceCreate_Returns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	apiErr := apierror.Parse(resp.Result())
	assert.Equal(t, "validation_failed", apiErr.Err.Code)
	assert.Contains(t, apiErr.Err.Fields, "provider_id")
	assert.Equal(t, "required", apiErr.Err.Fields["provider_id"].Code)
	assert.Contains(t, apiErr.Err.Fields, "type")
	assert.Equal(t, "required", apiErr.Err.Fields["type"].Code)
}

func Test_ServiceStart_WithAccessPolicy(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services?ignore_user_config=true",
		strings.NewReader(`{
			"type": "mockAccessPolicyService",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"access_policies": {
				"ids": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{
		priceToAdd: market.Price{
			PricePerHour: big.NewInt(500_000_000_000_000_000),
			PricePerGiB:  big.NewInt(1_000_000_000_000_000_000),
		},
	}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c9",
			"provider_id": "0xproviderid",
			"type": "mockAccessPolicyService",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
				"format": "service-proposal/v3",
				"compatibility": 2,
				"provider_id": "0xproviderid",
				"service_type": "mockAccessPolicyService",
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
                },
				"access_policies": [
					{
						"id":"verified-traffic",
						"source": "https://some.domain/api/v1/lists/verified-traffic"
					},
					{
						"id":"0x0000000000000001",
						"source": "https://some.domain/api/v1/lists/0x0000000000000001"
					},
					{
						"id":"dvpn-traffic",
						"source": "https://some.domain/api/v1/lists/dvpn-traffic"
					},
					{
						"id":"12312312332132",
						"source": "https://some.domain/api/v1/lists/12312312332132"
					}
				]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_ReturnsBadRequest_WithUnknownParams(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services",
		strings.NewReader(`{
			"type": "mockAccessPolicyService",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"access_policy": {
				"ids": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	g := summonTestGin()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{}, nil)(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "parse_failed", apierror.Parse(resp.Result()).Err.Code)
}
