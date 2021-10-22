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

	"github.com/gin-gonic/gin"

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
func (sm *mockServiceManager) List() map[service.ID]*service.Instance {
	return map[service.ID]*service.Instance{
		"11111111-9dad-11d1-80b4-00c04fd430c0": mockServiceStopped,
	}
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

func Test_AddRoutesForServiceAddsRoutes(t *testing.T) {
	router := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{
		priceToAdd: market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		},
	})(router)
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
				"id": "11111111-9dad-11d1-80b4-00c04fd430c0",
				"provider_id": "0xproviderid",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "NotRunning",
				"proposal": {
                    "format": "service-proposal/v3",
                    "compatibility": 1,
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
                      "bandwidth": 10
                    },
					"price": {
						"currency": "MYST",
						"per_gib": 2.0,
						"per_hour": 1.0
					}
				},
				"connection_statistics": {"attempted":0, "successful":0}
			}]`,
		},
		{
			http.MethodPost,
			"/services",
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
		            "compatibility": 1,
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
		              "bandwidth": 10
		            },
					"price": {
						"currency": "MYST",
						"per_gib": 2.0,
						"per_hour": 1.0
					}
				},
				"connection_statistics": {"attempted":0, "successful":0}
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
		            "compatibility": 1,
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
		              "bandwidth": 10
		            },
					"price": {
						"currency": "MYST",
						"per_gib": 2.0,
						"per_hour": 1.0
					}
				},
				"connection_statistics": {"attempted":0, "successful":0}
			}`,
		},
		{
			http.MethodDelete, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8", "",
			http.StatusAccepted, "",
		},
		{
			http.MethodDelete, "/services/00000000-9dad-11d1-80b4-00c04fd43000", "",
			http.StatusNotFound, `{"message":"Service not found"}`,
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

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"type": [ {"code": "invalid", "message": "Invalid service type"} ]
			}
		}`,
		resp.Body.String(),
	)
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

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"type": [ {"code": "invalid", "message": "Invalid service type"} ]
			}
		}`,
		resp.Body.String(),
	)
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

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"options": [ {"code": "invalid", "message": "Invalid options" } ]
			}
		}`,
		resp.Body.String(),
	)
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

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"Service already running"}`,
		resp.Body.String(),
	)
}

func Test_ServiceStatus_NotFoundIsReturnedWhenNotStarted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/services/1", nil)
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_ServiceGetReturnsServiceInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8", nil)
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForService(
		&mockServiceManager{},
		fakeOptionsParser,
		&mockProposalRepository{
			priceToAdd: market.Price{
				PricePerHour: big.NewInt(1),
				PricePerGiB:  big.NewInt(2),
			},
		},
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
				"compatibility": 1,
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
				  "bandwidth": 10
				},
				"price": {
					"currency": "MYST",
					"per_gib": 2.0,
					"per_hour": 1.0
				}
			},
			"connection_statistics": {"attempted":0, "successful":0}
		}`,
		resp.Body.String(),
	)
}
func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "invalid character 'a' looking for beginning of value"
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceCreate_Returns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(t,
		`{
			"message": "validation_error",
			"errors": {
				"provider_id": [ {"code": "required", "message": "Field is required"} ],
				"type": [ {"code": "required", "message": "Field is required"} ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_WithAccessPolicy(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/services",
		strings.NewReader(`{
			"type": "mockAccessPolicyService",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"access_policies": {
				"ids": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{
		priceToAdd: market.Price{
			PricePerHour: big.NewInt(1),
			PricePerGiB:  big.NewInt(2),
		},
	})(g)
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
				"compatibility": 1,
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
				  "bandwidth": 10
				},
				"price": {
					"currency": "MYST",
					"per_gib": 2.0,
					"per_hour": 1.0
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
			},
			"connection_statistics": {"attempted":0, "successful":0}
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

	g := gin.Default()
	err := AddRoutesForService(&mockServiceManager{}, fakeOptionsParser, &mockProposalRepository{})(g)
	assert.NoError(t, err)

	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{"message": "json: unknown field \"access_policy\""}`,
		resp.Body.String(),
	)
}
