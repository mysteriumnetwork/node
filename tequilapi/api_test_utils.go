/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package tequilapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestClient interface {
	Get(path string) *http.Response
}

type testClient struct {
	t       *testing.T
	baseURL string
}

// NewTestClient returns client for making test requests
func NewTestClient(t *testing.T, address string) TestClient {
	return &testClient{
		t,
		fmt.Sprintf("http://%s", address),
	}
}

func (tc *testClient) Get(path string) *http.Response {
	resp, err := http.Get(tc.baseURL + path)
	if err != nil {
		assert.FailNow(tc.t, "Uh oh catched error: ", err.Error())
	}
	return resp
}

func expectJSONStatus200(t *testing.T, resp *http.Response, httpStatus int) {
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-type"))
	assert.Equal(t, httpStatus, resp.StatusCode)
}

func parseResponseAsJSON(t *testing.T, resp *http.Response, v interface{}) {
	err := json.NewDecoder(resp.Body).Decode(v)
	assert.Nil(t, err)
}
