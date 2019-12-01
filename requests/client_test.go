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

package requests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientDoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpClient := NewHTTPClient("0.0.0.0", DefaultTimeout)

	req, err := NewGetRequest(server.URL, "/", nil)
	assert.NoError(t, err)

	res, err := httpClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestClientDoRequestAndParseResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Test": "OK"}`))
	}))
	defer server.Close()

	httpClient := NewHTTPClient("0.0.0.0", DefaultTimeout)

	req, err := NewGetRequest(server.URL, "/", nil)
	assert.NoError(t, err)

	var res struct {
		Test string
	}
	err = httpClient.DoRequestAndParseResponse(req, &res)
	assert.NoError(t, err)

	assert.Equal(t, "OK", res.Test)
}
