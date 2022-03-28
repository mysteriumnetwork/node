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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/requests"
)

func Test_TokenRewardAmount(t *testing.T) {
	mockResponse := `{"reward":"1000000000000000000","campaign_type":"consumer"}`
	server := newTestAffiliatorServer(http.StatusOK, mockResponse)

	router := summonTestGin()

	a := registry.NewAffiliator(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL)
	err := AddRoutesForAffiliator(a)(router)
	assert.NoError(t, err)
	//
	req, err := http.NewRequest(
		http.MethodGet,
		"/affiliator/token/mytoken/reward",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	expectedResponse := `{"amount":1000000000000000000}`
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, expectedResponse, resp.Body.String())
}

func newTestAffiliatorServer(mockStatus int, mockResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResponse))
	}))
}
