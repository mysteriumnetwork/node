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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockNATStatusProvider struct {
	mockStatus nat.Status
}

func (mockProvider *mockNATStatusProvider) Status() nat.Status {
	return mockProvider.mockStatus
}

func Test_NATStatus_ReturnsStatusSuccessful_WithSuccessfulEvent(t *testing.T) {
	testResponse(
		t,
		nat.Status{Status: statusSuccessful},
		`{
			"status": "successful"
		}`,
	)
}

func Test_NATStatus_ReturnsStatusFailureAndError_WithFailureEvent(t *testing.T) {
	testResponse(
		t,
		nat.Status{Status: statusFailure, Error: errors.New("mock error")},
		`{
			"status": "failure",
			"error": "mock error"
		}`,
	)
}

func Test_NATStatus_ReturnsStatusNotFinished_WhenEventIsNotAvailable(t *testing.T) {
	testResponse(
		t,
		nat.Status{Status: statusNotFinished},
		`{
			"status": "not_finished"
		}`,
	)
}

func testResponse(t *testing.T, mockStatus nat.Status, expectedJson string) {
	provider := mockNATStatusProvider{mockStatus: mockStatus}

	req, err := http.NewRequest(http.MethodGet, "/nat/status", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()
	router := httprouter.New()
	AddRoutesForNAT(router, provider.Status)

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, expectedJson, resp.Body.String())
}
