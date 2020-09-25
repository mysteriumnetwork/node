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

	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/identity"
	node_session "github.com/mysteriumnetwork/node/session"
)

var (
	connectionSessionMock = session.History{
		SessionID:       node_session.ID("ID"),
		ConsumerID:      identity.FromAddress("consumerID"),
		HermesID:        "0x000000000000000000000000000000000000000C",
		ProviderID:      identity.FromAddress("providerID"),
		ServiceType:     "serviceType",
		ConsumerCountry: "ConsumerCountry",
		ProviderCountry: "ProviderCountry",
		Started:         time.Date(2010, time.January, 1, 12, 00, 0, 700000000, time.UTC),
		Updated:         time.Date(2010, time.January, 1, 12, 00, 55, 800000000, time.UTC),
		DataSent:        10,
		DataReceived:    10,
	}
	sessionsMock = []session.History{
		connectionSessionMock,
	}
	sessionStatsMock = session.Stats{
		Count: 1,
	}
	sessionStatsByDayMock = map[time.Time]session.Stats{
		connectionSessionMock.Started: sessionStatsMock,
	}
)

func Test_SessionsEndpoint_SessionToDto(t *testing.T) {
	sessionDTO := contract.NewSessionDTO(connectionSessionMock)
	assert.Equal(t, "2010-01-01T12:00:00Z", sessionDTO.CreatedAt)
	assert.Equal(t, string(connectionSessionMock.SessionID), sessionDTO.ID)
	assert.Equal(t, connectionSessionMock.ConsumerID.Address, sessionDTO.ConsumerID)
	assert.Equal(t, connectionSessionMock.HermesID, sessionDTO.HermesID)
	assert.Equal(t, connectionSessionMock.ProviderID.Address, sessionDTO.ProviderID)
	assert.Equal(t, connectionSessionMock.ServiceType, sessionDTO.ServiceType)
	assert.Equal(t, connectionSessionMock.ConsumerCountry, sessionDTO.ConsumerCountry)
	assert.Equal(t, connectionSessionMock.ProviderCountry, sessionDTO.ProviderCountry)
	assert.Equal(t, connectionSessionMock.DataReceived, sessionDTO.BytesReceived)
	assert.Equal(t, connectionSessionMock.DataSent, sessionDTO.BytesSent)
	assert.Equal(t, 55, int(sessionDTO.Duration))
	assert.Equal(t, connectionSessionMock.Status, sessionDTO.Status)
}

func Test_SessionsEndpoint_List(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		sessionsToReturn:   sessionsMock,
		statsByDayToReturn: sessionStatsByDayMock,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := contract.SessionListResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(
		t,
		contract.SessionListResponse{
			Items: []contract.SessionDTO{
				contract.NewSessionDTO(connectionSessionMock),
			},
			PageableDTO: contract.PageableDTO{
				Page:       1,
				PageSize:   50,
				TotalItems: 1,
				TotalPages: 1,
			},
			StatsDaily: contract.NewSessionStatsDailyDTO(sessionStatsByDayMock),
		},
		parsedResponse,
	)
}

func Test_SessionsEndpoint_ListBubblesError(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	mockErr := errors.New("something exploded")
	ssm := &sessionStorageMock{
		errToReturn: mockErr,
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

func Test_SessionsEndpoint_Stats(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		statsToReturn: sessionStatsMock,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).Stats
	handlerFunc(resp, req, nil)

	parsedResponse := contract.SessionStatsDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(
		t,
		contract.NewSessionStatsDTO(sessionStatsMock),
		parsedResponse,
	)
}

type sessionStorageMock struct {
	sessionsToReturn   []session.History
	statsToReturn      session.Stats
	statsByDayToReturn map[time.Time]session.Stats
	errToReturn        error
}

func (ssm *sessionStorageMock) List(_ *session.Filter) ([]session.History, error) {
	return ssm.sessionsToReturn, ssm.errToReturn
}

func (ssm *sessionStorageMock) Stats(_ *session.Filter) (session.Stats, error) {
	return ssm.statsToReturn, ssm.errToReturn
}

func (ssm *sessionStorageMock) StatsByDay(_ *session.Filter) (map[time.Time]session.Stats, error) {
	return ssm.statsByDayToReturn, ssm.errToReturn
}
