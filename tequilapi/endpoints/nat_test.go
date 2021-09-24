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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/stretchr/testify/assert"
)

type mockNATProber struct {
	returnRes nat.NATType
	returnErr error
}

func (m *mockNATProber) Probe(_ context.Context) (nat.NATType, error) {
	return m.returnRes, m.returnErr
}

func Test_NATStatus_ReturnsStatusSuccessful_WithSuccessfulEvent(t *testing.T) {
	provider := &mockStateProvider{stateToReturn: stateEvent.State{
		NATStatus: contract.NATStatusDTO{
			Status: "something",
			Error:  "maybe",
		},
	}}
	natProber := &mockNATProber{
		returnRes: "none",
		returnErr: nil,
	}

	expectedJSON, err := json.Marshal(provider.stateToReturn.NATStatus)
	assert.Nil(t, err)

	req, err := http.NewRequest(http.MethodGet, "/nat/status", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()
	router := gin.Default()
	err = AddRoutesForNAT(provider, natProber)(router)
	assert.NoError(t, err)

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, string(expectedJSON), resp.Body.String())
}

func Test_NATStatus_ReturnsTypeSuccessful_WithSuccessfulEvent(t *testing.T) {
	provider := &mockStateProvider{stateToReturn: stateEvent.State{
		NATStatus: contract.NATStatusDTO{
			Status: "something",
			Error:  "maybe",
		},
	}}
	natProber := &mockNATProber{
		returnRes: "none",
		returnErr: nil,
	}

	expectedJSON, err := json.Marshal(contract.NATTypeDTO{
		Type:  natProber.returnRes,
		Error: "",
	})
	assert.Nil(t, err)

	req, err := http.NewRequest(http.MethodGet, "/nat/type", nil)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()
	router := gin.Default()
	err = AddRoutesForNAT(provider, natProber)(router)
	assert.NoError(t, err)

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, string(expectedJSON), resp.Body.String())
}
