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

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func Test_Docs_Index(t *testing.T) {
	endpoint, err := NewDocsEndpoint()
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()
	endpoint.Index(resp, req, httprouter.Params{})

	assert.Equal(t, 301, resp.Code)
}

func Test_Docs_Docs(t *testing.T) {
	endpoint, err := NewDocsEndpoint()
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()
	endpoint.Docs(resp, req, httprouter.Params{})

	assert.Equal(t, 200, resp.Code)
	assert.True(t, resp.Body.Len() > 1)
	assert.Contains(t, resp.Body.String(), `<redoc spec-url="tequilapi.json"></redoc>`)
}
