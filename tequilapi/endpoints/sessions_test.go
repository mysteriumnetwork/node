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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/go-rest/apierror"
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
		IPType:          "residential",
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
	assert.Equal(t, connectionSessionMock.IPType, sessionDTO.IPType)
}

func Test_SessionsEndpoint_List(t *testing.T) {
	url := "/sessions"
	req, err := http.NewRequest(
		http.MethodGet,
		url,
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		sessionsToReturn: sessionsMock,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).List

	g := summonTestGin()
	g.GET(url, handlerFunc)
	g.ServeHTTP(resp, req)

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
		},
		parsedResponse,
	)
	assert.Equal(t, session.NewFilter(), ssm.calledWithFilter)
}

func Test_SessionsEndpoint_ListRespectsFilters(t *testing.T) {
	path := "/sessions"
	ssm := &sessionStorageMock{
		sessionsToReturn: sessionsMock,
	}

	// when
	req, _ := http.NewRequest(
		http.MethodGet,
		path+"?date_from=2020-09-19&date_to=2020-09-20&direction=direction&service_type=service_type&status=status",
		nil,
	)
	resp := httptest.NewRecorder()
	g := summonTestGin()
	g.GET(path, NewSessionsEndpoint(ssm).List)
	g.ServeHTTP(resp, req)

	// then
	assert.Equal(
		t,
		session.NewFilter().
			SetStartedFrom(time.Date(2020, 9, 19, 0, 0, 0, 0, time.UTC)).
			SetStartedTo(time.Date(2020, 9, 20, 23, 59, 59, 0, time.UTC)).
			SetDirection("direction").
			SetServiceType("service_type").
			SetStatus("status"),
		ssm.calledWithFilter,
	)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func Test_SessionsEndpoint_ListBubblesError(t *testing.T) {
	path := "/sessions"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	mockErr := errors.New("something exploded")
	ssm := &sessionStorageMock{
		errToReturn: mockErr,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).List
	g := summonTestGin()
	g.GET(path, handlerFunc)
	g.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "err_session_list", apierror.Parse(resp.Result()).Err.Code)
}

func Test_SessionsEndpoint_StatsAggregated(t *testing.T) {
	path := "/sessions/stats-aggregated"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		statsToReturn: sessionStatsMock,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).StatsAggregated
	g := summonTestGin()
	g.GET(path, handlerFunc)
	g.ServeHTTP(resp, req)

	parsedResponse := contract.SessionStatsAggregatedResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(
		t,
		contract.SessionStatsAggregatedResponse{
			Stats: contract.NewSessionStatsDTO(sessionStatsMock),
		},
		parsedResponse,
	)
	assert.Equal(t, session.NewFilter(), ssm.calledWithFilter)
}

func Test_SessionsEndpoint_StatsDaily(t *testing.T) {
	path := "/sessions/stats-daily"
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	assert.Nil(t, err)

	ssm := &sessionStorageMock{
		statsToReturn:      sessionStatsMock,
		statsByDayToReturn: sessionStatsByDayMock,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewSessionsEndpoint(ssm).StatsDaily
	g := summonTestGin()
	g.GET(path, handlerFunc)
	g.ServeHTTP(resp, req)

	parsedResponse := contract.SessionStatsDailyResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(
		t,
		contract.SessionStatsDailyResponse{
			Items: map[string]contract.SessionStatsDTO{
				"2010-01-01": contract.NewSessionStatsDTO(sessionStatsMock),
			},
			Stats: contract.NewSessionStatsDTO(sessionStatsMock),
		},
		parsedResponse,
	)
	assert.NotEqual(t, session.NewFilter(), ssm.calledWithFilter)
	assert.Equal(t, time.Now().UTC().Add(-30*24*time.Hour).Day(), ssm.calledWithFilter.StartedFrom.Day())
	assert.Equal(t, time.Now().UTC().Day(), ssm.calledWithFilter.StartedTo.Day())
}

type sessionStorageMock struct {
	sessionsToReturn   []session.History
	statsToReturn      session.Stats
	statsByDayToReturn map[time.Time]session.Stats
	errToReturn        error

	calledWithFilter *session.Filter
}

func (ssm *sessionStorageMock) List(filter *session.Filter) ([]session.History, error) {
	ssm.calledWithFilter = filter
	return ssm.sessionsToReturn, ssm.errToReturn
}

func (ssm *sessionStorageMock) Stats(filter *session.Filter) (session.Stats, error) {
	ssm.calledWithFilter = filter
	return ssm.statsToReturn, ssm.errToReturn
}

func (ssm *sessionStorageMock) StatsByDay(filter *session.Filter) (map[time.Time]session.Stats, error) {
	ssm.calledWithFilter = filter
	return ssm.statsByDayToReturn, ssm.errToReturn
}
