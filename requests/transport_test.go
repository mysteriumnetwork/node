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
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		test          string
		calls         []*reqRes
		expectedError error
	}{
		{
			test: "Return success result without retries",
			calls: []*reqRes{
				{req: newMockRequest(), res: newMockResponse(), err: nil},
			},
		},
		{
			test: "Return success result after one retry",
			calls: []*reqRes{
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
				{req: newMockRequest(), res: newMockResponse(), err: nil},
			},
		},
		{
			test: "Return last response after max retries",
			calls: []*reqRes{
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
				{req: newMockRequest(), res: nil, err: errors.New("timeout error")},
			},
			expectedError: errors.New("timeout error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			transport := newTransport("0.0.0.0")
			transport.rt = &mockRoundTripper{calls: tt.calls}

			res, err := transport.RoundTrip(tt.calls[0].req)
			assert.Equal(t, tt.expectedError, err)
			if err == nil {
				assert.NotNil(t, res)
			}
		})
	}
}

func newMockRequest() *http.Request {
	req, _ := http.NewRequest("GET", "/", nil)
	return req
}

func newMockResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
	}
}

type reqRes struct {
	req *http.Request
	res *http.Response
	err error
}

type mockRoundTripper struct {
	index int
	calls []*reqRes
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	call := m.calls[m.index]
	m.index++
	return call.res, call.err
}
