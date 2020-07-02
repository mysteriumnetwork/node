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

	"github.com/mysteriumnetwork/node/consumer/session"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/stretchr/testify/assert"
)

var (
	serviceSessionMock = session.History{
		SessionID:  "session1",
		ConsumerID: identity.FromAddress("consumer1"),
		Started:    time.Now(),
	}
)

func Test_ServiceSessionsEndpoint_List(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	anotherSession := session.History{
		SessionID:  "session2",
		ConsumerID: identity.FromAddress("consumer1"),
		Started:    time.Now(),
	}

	ssm := &stateProviderMock{
		stateToReturn: stateEvent.State{
			Sessions: []session.History{
				serviceSessionMock,
				anotherSession,
			},
		},
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewServiceSessionsEndpoint(ssm).List
	handlerFunc(resp, req, nil)

	parsedResponse := &contract.ListConnectionSessionsResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, serviceSessionMock.ConsumerID.Address, parsedResponse.Sessions[0].ConsumerID)
	assert.Equal(t, string(serviceSessionMock.SessionID), parsedResponse.Sessions[0].ID)
	assert.Equal(t, serviceSessionMock.Started.Format(time.RFC3339), parsedResponse.Sessions[0].CreatedAt)

	assert.Equal(t, anotherSession.ConsumerID.Address, parsedResponse.Sessions[1].ConsumerID)
	assert.Equal(t, string(anotherSession.SessionID), parsedResponse.Sessions[1].ID)
	assert.Equal(t, anotherSession.Started.Format(time.RFC3339), parsedResponse.Sessions[1].CreatedAt)

}

type stateProviderMock struct {
	stateToReturn stateEvent.State
}

func (spm *stateProviderMock) GetState() stateEvent.State {
	return spm.stateToReturn
}
