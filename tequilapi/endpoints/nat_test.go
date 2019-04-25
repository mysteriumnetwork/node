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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_NatStatus_ReturnsSuccess_WithSuccessfulEvent(t *testing.T) {
	successfulEvent := traversal.BuildSuccessEvent("hole_punching")

	testResponse(
		t,
		natTrackerMock{mockLastEvent: &successfulEvent},
		`{
			"successful": true,
			"error": ""
		}`,
	)
}

func Test_NatStatus_ReturnsFailureWithError_WithFailureEvent(t *testing.T) {
	failureEvent := traversal.BuildFailureEvent("hole_punching", errors.New("mock error"))

	testResponse(
		t,
		natTrackerMock{mockLastEvent: &failureEvent},
		`{
			"successful": false,
			"error": "mock error"
		}`,
	)
}

func Test_NatStatus_ReturnsError_WhenEventIsNotAvailable(t *testing.T) {
	testErrorResponse(
		t,
		natTrackerMock{mockLastEvent: nil},
		`{
			"message": "No status is available"
		}`,
	)
}

func testResponse(t *testing.T, mockedTracker natTrackerMock, expectedJson string) {
	resp, err := makeStatusRequestAndReturnResponse(mockedTracker)
	assert.Nil(t, err)

	parsedResponse := &NatStatusDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.JSONEq(t, expectedJson, resp.Body.String())
}

func testErrorResponse(t *testing.T, mockedTracker natTrackerMock, expectedJson string) {
	resp, err := makeStatusRequestAndReturnResponse(mockedTracker)
	assert.Nil(t, err)

	assert.JSONEq(t, expectedJson, resp.Body.String())
}

func makeStatusRequestAndReturnResponse(mockedTracker natTrackerMock) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest(http.MethodGet, "/nat/status", nil)
	if err != nil {
		return nil, err
	}

	resp := httptest.NewRecorder()
	router := httprouter.New()
	AddRoutesForNat(router, &mockedTracker)
	router.ServeHTTP(resp, req)

	return resp, nil
}

type natTrackerMock struct {
	mockLastEvent *traversal.Event
}

func (nt *natTrackerMock) LastEvent() *traversal.Event {
	return nt.mockLastEvent
}
