/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/assert"
)

func Test_Docs(t *testing.T) {
	// given
	router := gin.Default()
	err := AddRoutesForDocs(router)
	assert.NoError(t, err)

	// when
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// then
	assert.Equal(t, 301, resp.Code)
	assert.Equal(t, "/docs", resp.Header().Get("Location"))

	// when
	req, _ = http.NewRequest("GET", "/docs", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	// then
	assert.Equal(t, 301, resp.Code)
	assert.Equal(t, "/docs/", resp.Header().Get("Location"))

	// when
	req, _ = http.NewRequest("GET", "/docs/", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	// then
	assert.Equal(t, 200, resp.Code)
	assert.Contains(t, resp.Body.String(), `<redoc spec-url="./swagger.json"></redoc>`)

	// when
	req, _ = http.NewRequest("GET", "/docs/index.html", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	// then
	assert.Equal(t, 301, resp.Code)
	assert.Equal(t, "./", resp.Header().Get("Location"))

	// when
	req, _ = http.NewRequest("GET", "/docs/swagger.json", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	// then
	assert.Equal(t, 200, resp.Code)
	assert.Contains(t, resp.Body.String(), `"host": "127.0.0.1:4050"`)
}
