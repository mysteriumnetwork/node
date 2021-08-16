/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

type fakeEthClient struct {
	urls []string
}

func newFakeEthClient(urls []string) *fakeEthClient {
	return &fakeEthClient{urls: urls}
}

func (f *fakeEthClient) CurrentClientOrder() []string {
	return f.urls
}

func Test_EthEndpoints(t *testing.T) {
	endpoint := newEthInfoEndpoint(newFakeEthClient([]string{"one", "two"}))
	req := httptest.NewRequest(
		http.MethodPut,
		"/irrelevant",
		strings.NewReader(
			`{
				"consumer_id" : "my-identity",
				"provider_id" : "required-node",
				"hermes_id" : "hermes"
			}`))
	resp := httptest.NewRecorder()

	endpoint.EthInfo(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{"eth_rpc_l2_url":"one"}`,
		resp.Body.String(),
	)
}
