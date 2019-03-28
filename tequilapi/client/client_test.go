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

package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const errorMessage = `
{
	"message" : "me haz faild"
}
`

func TestConnectionErrorIsReturnedByClientInsteadOfDoubleParsing(t *testing.T) {
	responseBody := &trackingCloser{
		Reader: strings.NewReader(errorMessage),
	}

	client := Client{
		http: &httpClient{
			http: onAnyRequestReturn(&http.Response{
				Status:     "Internal server error",
				StatusCode: 500,
				Body:       responseBody,
			}),
			baseURL:   "http://test-api-whatever",
			logPrefix: "test prefix ",
			ua:        "test-agent",
		},
	}

	_, err := client.Connect("consumer", "provider", "service", ConnectOptions{})
	assert.Error(t, err)
	assert.Equal(t, errors.New("server response invalid: Internal server error (http://test-api-whatever/connection). Possible error: me haz faild"), err)
	//when doing http request, response body should always be closed by client - otherwise persistent connections are leaking
	assert.True(t, responseBody.Closed)
}

type requestDoer func(req *http.Request) (*http.Response, error)

func (f requestDoer) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func onAnyRequestReturn(response *http.Response) requestDoer {
	return func(req *http.Request) (*http.Response, error) {
		response.Request = req
		return response, nil
	}
}

type trackingCloser struct {
	io.Reader
	Closed bool
}

func (tc *trackingCloser) Close() error {
	tc.Closed = true
	return nil
}

var _ io.ReadCloser = (*trackingCloser)(nil)
