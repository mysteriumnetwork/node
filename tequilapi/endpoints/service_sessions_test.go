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
	"time"

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

	anotherSession := session.Session{
		ID:         session.ID("session2"),
		ConsumerID: identity.FromAddress("consumer1"),
		CreatedAt:  time.Now(),
	}

	ssm := &serviceSessionStorageMock{
		sessionsToReturn: []session.Session{
			serviceSessionMock,
			anotherSession,
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewServiceSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &serviceSessionsList{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, serviceSessionToDto(serviceSessionMock), parsedResponse.Sessions[0])
	assert.EqualValues(t, serviceSessionToDto(anotherSession), parsedResponse.Sessions[1])
}

type serviceSessionStorageMock struct {
	sessionsToReturn []session.Session
}

func (ssm *serviceSessionStorageMock) GetAll() []session.Session {
	return ssm.sessionsToReturn
}
