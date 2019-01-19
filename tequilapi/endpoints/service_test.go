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
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

type fakeServiceManager struct {
}

func (fm *fakeServiceManager) Start(providerID identity.Identity, serviceType string, options service.Options) (err error) {
	return nil
}

func (fm *fakeServiceManager) Kill() error {
	return nil
}

func TestAddRoutesForServiceAddsRoutes(t *testing.T) {
	router := httprouter.New()
	AddRoutesForService(router, &service.Manager{})

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet, "/service", "",
			http.StatusOK, `{"status": "NotRunning"}`,
		},
		{
			http.MethodPut, "/service", `{"providerId": "node1", "serviceType": "noop"}`,
			http.StatusCreated, `{"status": "Running"}`,
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

func Test_ServiceStatus_NotRunningStateIsReturnedWhenNotStarted(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&fakeServiceManager{})

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.Status(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
            "status" : "NotRunning"
        }`,
		resp.Body.String(),
	)
}

func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&fakeServiceManager{})

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	serviceEndpoint.Create(resp, req, httprouter.Params{})

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
	serviceEndpoint := NewServiceEndpoint(&fakeServiceManager{})

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	serviceEndpoint.Create(resp, req, httprouter.Params{})

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
