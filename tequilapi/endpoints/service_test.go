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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	mockServiceID      = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	mockACLServiceID   = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c9")
	mockServiceOptions = fancyServiceOptions{
		Foo: "bar",
	}
	mockACLEndpoint = "https://some.domain/api/v1/lists/"
	mockProposal    = market.ServiceProposal{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
	}
	acls = []market.ACL{
		{
			Protocol: "http",
			ListIds:  []string{"verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"},
			Links: market.ACLLinks{
				List: market.ACLList{
					Href: fmt.Sprintf("%v{ref}", mockACLEndpoint),
				},
			},
		},
	}
	serviceTypeWithACL = "mockaclservice"
	mockACLProposal    = market.ServiceProposal{
		ID:                1,
		ServiceType:       serviceTypeWithACL,
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
		ACL:               &acls,
	}
	mockServiceRunning        = service.NewInstance(mockServiceOptions, service.Running, nil, mockProposal, nil, nil)
	mockServiceStopped        = service.NewInstance(mockServiceOptions, service.NotRunning, nil, mockProposal, nil, nil)
	mockServiceRunningWithACL = service.NewInstance(mockServiceOptions, service.Running, nil, mockACLProposal, nil, nil)
)

type fancyServiceOptions struct {
	Foo string `json:"foo"`
}

type mockServiceManager struct{}

func (sm *mockServiceManager) Start(providerID identity.Identity, serviceType string, acl *[]market.ACL, options service.Options) (service.ID, error) {
	if serviceType == serviceTypeWithACL {
		return mockACLServiceID, nil
	}
	return mockServiceID, nil
}
func (sm *mockServiceManager) Stop(id service.ID) error { return nil }
func (sm *mockServiceManager) Service(id service.ID) *service.Instance {
	if id == "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		return mockServiceRunning
	}
	if id == mockACLServiceID {
		return mockServiceRunningWithACL
	}
	return nil
}
func (sm *mockServiceManager) List() map[service.ID]*service.Instance {
	return map[service.ID]*service.Instance{
		"11111111-9dad-11d1-80b4-00c04fd430c0": mockServiceStopped,
	}
}
func (sm *mockServiceManager) Kill() error { return nil }

var fakeOptionsParser = map[string]ServiceOptionsParser{
	"testprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	serviceTypeWithACL: func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	"errorprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, errors.New("error")
	},
}

func Test_AddRoutesForServiceAddsRoutes(t *testing.T) {
	router := httprouter.New()
	AddRoutesForService(router, &mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

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
				"providerId": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "NotRunning",
				"proposal": {
					"id": 1,
					"providerId": "0xProviderId",
					"serviceType": "testprotocol",
					"serviceDefinition": {
						"locationOriginate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
					}
				}
			}]`,
		},
		{
			http.MethodPost,
			"/services",
			`{"providerId": "node1", "type": "testprotocol"}`,
			http.StatusCreated,
			`{
				"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				"providerId": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
					"id": 1,
					"providerId": "0xProviderId",
					"serviceType": "testprotocol",
					"serviceDefinition": {
						"locationOriginate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
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
				"providerId": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
					"id": 1,
					"providerId": "0xProviderId",
					"serviceType": "testprotocol",
					"serviceDefinition": {
						"locationOriginate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
					}
				}
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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "openvpn",
			"providerId": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "openvpn",
			"providerId": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "errorprotocol",
			"providerId": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "testprotocol",
			"providerId": "0xProviderId",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"Service already running"}`,
		resp.Body.String(),
	)
}

func Test_ServiceStatus_NotFoundIsReturnedWhenNotStarted(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_ServiceGetReturnsServiceInfo(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{{Key: "id", Value: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"}})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"providerId": "0xProviderId",
			"type": "testprotocol",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
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
		}`,
		resp.Body.String(),
	)
}
func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(t,
		`{
			"message": "validation_error",
			"errors": {
				"providerId": [ {"code": "required", "message": "Field is required"} ],
				"type": [ {"code": "required", "message": "Field is required"} ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_WithACL(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser, mockACLEndpoint)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "mockaclservice",
			"providerId": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"acl": {
				"listIds": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c9",
			"providerId": "0xProviderId",
			"type": "mockaclservice",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
				"id": 1,
				"providerId": "0xProviderId",
				"serviceType": "mockaclservice",
				"serviceDefinition": {
					"locationOriginate": {"asn": "LT", "country": "Lithuania", "city": "Vilnius"}
				},
				"acl": [
					{
						"_links": {
							"list": {
								"href": "https://some.domain/api/v1/lists/{ref}"
							}
						},
						"listIds": [
							"verified-traffic",
							"dvpn-traffic",
							"12312312332132",
							"0x0000000000000001"
						],
						"protocol": "http"
					}
				]
			}
		}`,
		resp.Body.String(),
	)
}
