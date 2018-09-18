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

package server

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

var testRequestAPIURL = "http://testUrl"

func TestHttpErrorIsReportedAsErrorReturnValue(t *testing.T) {
	req, err := requests.NewGetRequest(context.Background(), testRequestAPIURL, "path", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: 400,
		Request:    req,
	}
	err = parseResponseError(response)
	assert.Error(t, err)
}

type testResponse struct {
	MyValue string `json:"myValue"`
}

func TestHttpResponseBodyIsParsedCorrectly(t *testing.T) {

	req, err := requests.NewGetRequest(context.Background(), testRequestAPIURL, "path", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: 200,
		Request:    req,
		Body: ioutil.NopCloser(bytes.NewBufferString(
			`{
				"myValue" : "abc"
			}`)),
	}
	var testDto testResponse
	err = parseResponseJSON(response, &testDto)
	assert.NoError(t, err)
	assert.Equal(t, testResponse{"abc"}, testDto)

}
