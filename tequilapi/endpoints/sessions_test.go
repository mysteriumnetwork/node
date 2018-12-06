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

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

var (
	sessionID        = node_session.ID("SessionID")
	ProviderID       = identity.FromAddress("ProviderID")
	ServiceType      = "ServiceType"
	ProviderCountry  = "ProviderCountry"
	SessionStatsMock = consumer.SessionStatistics{
		BytesReceived: 10,
		BytesSent:     10,
	}
	SessionMock = session.History{
		SessionID:       sessionID,
		ProviderID:      ProviderID,
		ServiceType:     ServiceType,
		ProviderCountry: ProviderCountry,
		Started:         time.Now(),
		Updated:         time.Now(),
		DataStats:       SessionStatsMock,
	}
)

func TestSessionToDto(t *testing.T) {
	value := "2010-01-01T12:00:00Z"
	startedAt, _ := time.Parse(time.RFC3339, value)

	se := SessionMock
	se.Started = startedAt
	sessionDTO := toHistoryView(se)

	assert.Equal(t, value, sessionDTO.DateStarted)
	assert.Equal(t, string(SessionMock.SessionID), sessionDTO.SessionID)
	assert.Equal(t, SessionMock.ProviderID.Address, sessionDTO.ProviderID)
	assert.Equal(t, SessionMock.ServiceType, sessionDTO.ServiceType)
	assert.Equal(t, SessionMock.ProviderCountry, sessionDTO.ProviderCountry)
	assert.Equal(t, SessionMock.DataStats.BytesReceived, sessionDTO.BytesReceived)
	assert.Equal(t, SessionMock.DataStats.BytesSent, sessionDTO.BytesSent)
	assert.Equal(t, SessionMock.GetDuration(), sessionDTO.Duration)
	assert.Equal(t, SessionMock.Status, sessionDTO.Status)
}

func TestListEndpoint(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		errToReturn: nil,
		sessionsToReturn: []session.History{
			SessionMock,
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &SessionsDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, toHistoryView(SessionMock), parsedResponse.Sessions[0])
}

func TestListEndpointBubblesError(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	mockErr := errors.New("something exploded")
	ssm := &sessionStorageMock{
		errToReturn:      mockErr,
		sessionsToReturn: []session.History{},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t,
		fmt.Sprintf(`{"message":%q}%v`, mockErr.Error(), "\n"),
		resp.Body.String(),
	)
}

type sessionStorageMock struct {
	sessionsToReturn []session.History
	errToReturn      error
}

func (ssm *sessionStorageMock) GetAll() ([]session.History, error) {
	return ssm.sessionsToReturn, ssm.errToReturn
}
