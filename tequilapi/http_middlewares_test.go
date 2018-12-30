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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorsHeadersAreAppliedToResponse(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/not-important", nil)
	assert.NoError(t, err)

	respRecorder := httptest.NewRecorder()

	mock := &mockedHTTPHandler{}

	ApplyCors(mock).ServeHTTP(respRecorder, req)

	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Methods"))
	assert.True(t, mock.wasCalled)
}

func TestPreflightCorsCheckIsHandled(t *testing.T) {
	req, err := http.NewRequest(http.MethodOptions, "/not-important", nil)
	assert.NoError(t, err)
	req.Header.Add("Origin", "Original site")
	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("Access-Control-Request-Headers", "origin, x-requested-with")

	respRecorder := httptest.NewRecorder()

	mock := &mockedHTTPHandler{}

	ApplyCors(mock).ServeHTTP(respRecorder, req)

	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "origin, x-requested-with", respRecorder.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, 0, respRecorder.Body.Len())
	assert.False(t, mock.wasCalled)
}

func TestDeleteCorsPreflightCheckIsHandledCorrectly(t *testing.T) {
	req, err := http.NewRequest(http.MethodOptions, "/not-important", nil)
	assert.NoError(t, err)
	req.Header.Add("Origin", "Original site")
	req.Header.Add("Access-Control-Request-Method", "DELETE")

	respRecorder := httptest.NewRecorder()

	mock := &mockedHTTPHandler{}

	ApplyCors(mock).ServeHTTP(respRecorder, req)

	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, respRecorder.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, 0, respRecorder.Body.Len())
	assert.False(t, mock.wasCalled)

}

func TestCacheControlHeadersAreAddedToResponse(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/not-important", nil)
	assert.NoError(t, err)
	respRecorder := httptest.NewRecorder()

	mock := &mockedHTTPHandler{}

	DisableCaching(mock).ServeHTTP(respRecorder, req)

	assert.Equal(
		t,
		"no-cache, no-store, must-revalidate",
		respRecorder.Header().Get("Cache-Control"),
	)
	assert.True(t, mock.wasCalled)

}

type mockedHTTPHandler struct {
	wasCalled bool
}

func (mock *mockedHTTPHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	mock.wasCalled = true
}
