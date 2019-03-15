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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

var (
	serviceSessionMock = session.Session{
		ID:         session.ID("session1"),
		ConsumerID: identity.FromAddress("consumer1"),
	}
)

func Test_ServiceSessionsEndpoint_SessionToDto(t *testing.T) {
	se := serviceSessionMock
	sessionDTO := serviceSessionToDto(se)

	assert.Equal(t, string(serviceSessionMock.ID), sessionDTO.ID)
	assert.Equal(t, serviceSessionMock.ConsumerID.Address, sessionDTO.ConsumerID)
}

func Test_ServiceSessionsEndpoint_List(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	ssm := &serviceSessionStorageMock{
		errToReturn: nil,
		sessionsToReturn: []session.Session{
			serviceSessionMock,
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewServiceSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &serviceSessionsList{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, serviceSessionToDto(serviceSessionMock), parsedResponse.Sessions[0])
}

func Test_ServiceSessionsEndpoint_ListBubblesError(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	mockErr := errors.New("something exploded")
	ssm := &serviceSessionStorageMock{
		errToReturn:      mockErr,
		sessionsToReturn: []session.Session{},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewServiceSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t,
		fmt.Sprintf(`{"message":%q}%v`, mockErr.Error(), "\n"),
		resp.Body.String(),
	)
}

type serviceSessionStorageMock struct {
	sessionsToReturn []session.Session
	errToReturn      error
}

func (ssm *serviceSessionStorageMock) GetAll() ([]session.Session, error) {
	return ssm.sessionsToReturn, ssm.errToReturn
}
