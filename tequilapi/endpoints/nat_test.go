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
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	emptyEvent = traversal.Event{
		Type:  "",
		Error: errors.New("Some message"),
	}
	failedEvent = traversal.Event{
		Type:  "failure",
		Error: errors.New("no UPnP or NAT-PMP router discovered"),
	}
	successfulEvent = traversal.Event{
		Type:  "success",
		Error: errors.New(""),
	}
)

func Test_NatEndpoint_Status_WhenStatusIsNotAvailable(t *testing.T) {
	expected := errorMessage{Message: "No status is available"}

	testErrorResponse(t, natTrackerMock{hasNoEvent: true}, &expected)
}

func Test_NatEndpoint_Status_WhenStatusIsFailed(t *testing.T) {
	expected := toNatStatusResponse(failedEvent)

	testResponse(t, natTrackerMock{}, &expected)
}

func Test_NatEndpoint_Status_WhenStatusIsSuccessful(t *testing.T) {
	expected := toNatStatusResponse(successfulEvent)

	testResponse(t, natTrackerMock{hasSuccessfulEvent: true}, &expected)
}

func testErrorResponse(t *testing.T, mockedTracker natTrackerMock, expected interface{}) {
	resp, err := makeStatusRequestAndReturnResponse(t, mockedTracker)
	assert.Nil(t, err)

	parsedResponse := &errorMessage{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, expected, parsedResponse)
}

func testResponse(t *testing.T, mockedTracker natTrackerMock, expected interface{}) {
	resp, err := makeStatusRequestAndReturnResponse(t, mockedTracker)
	assert.Nil(t, err)

	parsedResponse := &NatStatusDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, expected, parsedResponse)
}

func makeStatusRequestAndReturnResponse(t *testing.T, mockedTracker natTrackerMock) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewNatEndpoint(&mockedTracker).NatStatus
	handlerFunc(resp, req, nil)

	return resp, nil
}

type natTrackerMock struct {
	hasSuccessfulEvent bool
	hasNoEvent         bool
}

type errorMessage struct {
	Message string `json:"message"`
}

func (nt *natTrackerMock) LastEvent() traversal.Event {
	if nt.hasSuccessfulEvent {
		return successfulEvent
	}

	if nt.hasNoEvent {
		return emptyEvent
	}

	return failedEvent
}
