/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

var (
	connectionSessionMock = session.History{
		SessionID:       node_session.ID("SessionID"),
		ConsumerID:      identity.FromAddress("consumerID"),
		AccountantID:    "0x000000000000000000000000000000000000000C",
		ProviderID:      identity.FromAddress("providerID"),
		ServiceType:     "serviceType",
		ProviderCountry: "ProviderCountry",
		Started:         time.Date(2010, time.January, 1, 12, 00, 0, 700000000, time.UTC),
		Updated:         time.Date(2010, time.January, 1, 12, 00, 55, 800000000, time.UTC),
		DataStats: connection.Statistics{
			BytesReceived: 10,
			BytesSent:     10,
		},
	}
)

func Test_ConnectionSessionsEndpoint_SessionToDto(t *testing.T) {
	sessionDTO := connectionSessionToDto(connectionSessionMock)
	assert.Equal(t, "2010-01-01T12:00:00Z", sessionDTO.DateStarted)
	assert.Equal(t, string(connectionSessionMock.SessionID), sessionDTO.SessionID)
	assert.Equal(t, connectionSessionMock.ConsumerID.Address, sessionDTO.ConsumerID)
	assert.Equal(t, connectionSessionMock.AccountantID, sessionDTO.AccountantID)
	assert.Equal(t, connectionSessionMock.ProviderID.Address, sessionDTO.ProviderID)
	assert.Equal(t, connectionSessionMock.ServiceType, sessionDTO.ServiceType)
	assert.Equal(t, connectionSessionMock.ProviderCountry, sessionDTO.ProviderCountry)
	assert.Equal(t, connectionSessionMock.DataStats.BytesReceived, sessionDTO.BytesReceived)
	assert.Equal(t, connectionSessionMock.DataStats.BytesSent, sessionDTO.BytesSent)
	assert.Equal(t, 55, int(sessionDTO.Duration))
	assert.Equal(t, connectionSessionMock.Status, sessionDTO.Status)
}

func Test_ConnectionSessionsEndpoint_List(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	ssm := &connectionSessionStorageMock{
		errToReturn: nil,
		sessionsToReturn: []session.History{
			connectionSessionMock,
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewConnectionSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &connectionSessionsList{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, connectionSessionToDto(connectionSessionMock), parsedResponse.Sessions[0])
}

func Test_ConnectionSessionsEndpoint_ListBubblesError(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	mockErr := errors.New("something exploded")
	ssm := &connectionSessionStorageMock{
		errToReturn:      mockErr,
		sessionsToReturn: []session.History{},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewConnectionSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t,
		fmt.Sprintf(`{"message":%q}%v`, mockErr.Error(), "\n"),
		resp.Body.String(),
	)
}

type connectionSessionStorageMock struct {
	sessionsToReturn []session.History
	errToReturn      error
}

func (ssm *connectionSessionStorageMock) GetAll() ([]session.History, error) {
	return ssm.sessionsToReturn, ssm.errToReturn
}
