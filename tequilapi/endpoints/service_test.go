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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var mockServiceID = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

type mockServiceManager struct{}

func (sm *mockServiceManager) Start(providerID identity.Identity, serviceType string, options service.Options) (service.ID, error) {
	return mockServiceID, nil
}
func (sm *mockServiceManager) Stop(id service.ID) error { return nil }
func (sm *mockServiceManager) Service(id service.ID) *service.Instance {
	if id == "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		return service.NewInstance(service.Running, nil, serviceProposals[0], nil, nil)
	}
	return nil
}
func (sm *mockServiceManager) List() map[service.ID]*service.Instance {
	return map[service.ID]*service.Instance{
		"11111111-9dad-11d1-80b4-00c04fd430c0": service.NewInstance(service.NotRunning, nil, serviceProposals[0], nil, nil),
	}
}
func (sm *mockServiceManager) Kill() error { return nil }

var fakeOptionsParser = map[string]func(json.RawMessage) (service.Options, error){
	"testprotocol": func(opts json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	"errorprotocol": func(opts json.RawMessage) (service.Options, error) {
		return nil, errors.New("error")
	},
}

func Test_AddRoutesForServiceAddsRoutes(t *testing.T) {
	router := httprouter.New()
	AddRoutesForService(router, &mockServiceManager{}, fakeOptionsParser)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/services", "",
			http.StatusOK, `[{"id":"11111111-9dad-11d1-80b4-00c04fd430c0","proposal":{"id":1,"providerId":"0xProviderId","serviceType":"testprotocol","serviceDefinition":{"locationOriginate":{"asn":"LT","country":"Lithuania","city":"Vilnius"}}},"status":"NotRunning","options":{"protocol":"","port":0}}]`,
		},
		{
			http.MethodPost, "/services", `{"providerId": "node1", "serviceType": "testprotocol"}`,
			http.StatusCreated, `{"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","proposal":{"id":1,"providerId":"0xProviderId","serviceType":"testprotocol","serviceDefinition":{"locationOriginate":{"asn":"LT","country":"Lithuania","city":"Vilnius"}}},"status":"Running","options":{"protocol":"","port":0}}`,
		},
		{
			http.MethodGet, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8", "",
			http.StatusOK, `{"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","proposal":{"id":1,"providerId":"0xProviderId","serviceType":"testprotocol","serviceDefinition":{"locationOriginate":{"asn":"LT","country":"Lithuania","city":"Vilnius"}}},"status":"Running","options":{"protocol":"","port":0}}`,
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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", strings.NewReader(`{"serviceType":"openvpn","providerId":"0x9edf75f870d87d2d1a69f0d950a99984ae955ee0","options":{"openvpnPort":1123,"openvpnProtocol":"UDP"}}`))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"Invalid service type"}`,
		resp.Body.String(),
	)
}

func Test_ServiceStartInvalidOptions(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", strings.NewReader(`{"serviceType":"errorprotocol","providerId":"0x9edf75f870d87d2d1a69f0d950a99984ae955ee0","options":{"openvpnPort":1123,"openvpnProtocol":"UDP"}}`))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"Invalid options"}`,
		resp.Body.String(),
	)
}

func Test_ServiceStartAlreadyRunning(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", strings.NewReader(`{"serviceType":"testprotocol","providerId":"0xProviderId","options":{"openvpnPort":1123,"openvpnProtocol":"UDP"}}`))
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
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_ServiceGetReturnsServiceInfo(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{{Key: "id", Value: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"}})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","proposal":{"id":1,"providerId":"0xProviderId","serviceType":"testprotocol","serviceDefinition":{"locationOriginate":{"asn":"LT","country":"Lithuania","city":"Vilnius"}}},"status":"Running","options":{"protocol":"","port":0}}`,
		resp.Body.String(),
	)
}
func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message" : "invalid character 'a' looking for beginning of value"
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceCreate_Returns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(t,
		`{
			"message" : "validation_error",
			"errors" : {
				"providerId" : [ {"code" : "required" , "message" : "Field is required" } ],
				"serviceType" : [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}
