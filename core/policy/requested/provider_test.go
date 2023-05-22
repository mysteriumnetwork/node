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

package requested_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/policy/requested"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

func Test_RequestedProvider_GetIdentityAllowed(t *testing.T) {
	policyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("identity-value") {
		case "0x1":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "1",
				"title": "One",
				"description": "",
				"allow": [
					{"type": "identity", "value": "0x1"}
				]
			}`))
		case "0x2":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "2",
				"title": "Two",
				"description": "",
				"allow": [
					{"type": "identity", "value": "0x2"}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer policyServer.Close()

	oracle := requested.NewRequestedProvider(requests.NewHTTPClient("0.0.0.0", 100*time.Millisecond), policyServer.URL+"/")

	assert.True(t, oracle.IsIdentityAllowed(identity.Identity{Address: "0x1"}))
	assert.True(t, oracle.IsIdentityAllowed(identity.Identity{Address: "0x2"}))
	assert.False(t, oracle.IsIdentityAllowed(identity.Identity{Address: "0x3"}))
}
