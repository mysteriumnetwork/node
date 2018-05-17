/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type handleFunctionTestStruct struct {
	called bool
}

func (hfts *handleFunctionTestStruct) httprouterHandle(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	hfts.called = true
}

func TestHttpRouterHandlesRequests(t *testing.T) {
	ts := handleFunctionTestStruct{false}

	router := httprouter.New()
	router.GET("/testhandler", ts.httprouterHandle)

	req, err := http.NewRequest("GET", "/testhandler", nil)
	assert.Nil(t, err)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)
	assert.Equal(t, true, ts.called)
}
