/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type MockAuthenticator struct {
	Err error
}

func (m *MockAuthenticator) ValidateToken(token string) (bool, error) {
	return false, m.Err
}

func (m *MockAuthenticator) SetErr(err error) {
	m.Err = err
}

func TestTokenParsing(t *testing.T) {
	// given
	authenticator := &MockAuthenticator{Err: errors.New("")}

	req, err := http.NewRequest(http.MethodGet, "/not-important", nil)
	assert.NoError(t, err)
	respRecorder := httptest.NewRecorder()

	g := gin.Default()
	g.Use(ApplyMiddlewareTokenAuth(authenticator))

	// expect
	g.ServeHTTP(respRecorder, req)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		respRecorder.Code,
	)

	//and
	authenticator.SetErr(nil)
	respRecorder = httptest.NewRecorder()
	g.ServeHTTP(respRecorder, req)
	assert.Equal(
		t,
		http.StatusNotFound,
		respRecorder.Code,
	)

	//and
	authenticator.SetErr(nil)
	req, err = http.NewRequest(http.MethodGet, "/not-important", nil)
	assert.NoError(t, err)
	respRecorder = httptest.NewRecorder()

	// no 'Bearer' prefix
	req.Header = map[string][]string{"Authorization": {"123"}}

	g.ServeHTTP(respRecorder, req)
	assert.Equal(
		t,
		http.StatusBadRequest,
		respRecorder.Code,
	)
}
