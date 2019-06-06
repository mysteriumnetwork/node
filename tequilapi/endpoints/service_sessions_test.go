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

	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/stretchr/testify/assert"
)

var (
	serviceSessionMock = stateEvent.ServiceSession{
		ID:         "session1",
		ConsumerID: "consumer1",
		CreatedAt:  time.Now(),
	}
)

func Test_ServiceSessionsEndpoint_List(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	anotherSession := stateEvent.ServiceSession{
		ID:         "session2",
		ConsumerID: "consumer1",
		CreatedAt:  time.Now(),
	}

	ssm := &stateProviderMock{
		stateToReturn: stateEvent.State{
			Sessions: []stateEvent.ServiceSession{
				serviceSessionMock,
				anotherSession,
			},
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewServiceSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &serviceSessionsList{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, serviceSessionMock.ConsumerID, parsedResponse.Sessions[0].ConsumerID)
	assert.Equal(t, serviceSessionMock.ID, parsedResponse.Sessions[0].ID)
	assert.True(t, serviceSessionMock.CreatedAt.Equal(parsedResponse.Sessions[0].CreatedAt))

	assert.Equal(t, anotherSession.ConsumerID, parsedResponse.Sessions[1].ConsumerID)
	assert.Equal(t, anotherSession.ID, parsedResponse.Sessions[1].ID)
	assert.True(t, anotherSession.CreatedAt.Equal(parsedResponse.Sessions[1].CreatedAt))

}

type stateProviderMock struct {
	stateToReturn stateEvent.State
}

func (spm *stateProviderMock) GetState() stateEvent.State {
	return spm.stateToReturn
}
