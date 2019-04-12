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

package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func Test_Get_ACL(t *testing.T) {
	router := httprouter.New()
	AddRoutesForACL(router)

	req, err := http.NewRequest(
		http.MethodGet,
		"/acl",
		nil,
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"entries": [
				{
					"name": "mysterium",
					"description": "Mysterium Network approved identities",
					"allow": [
						{
							"type": "identity",
							"value": "0xf4d6ffba09d460ebe10d24667770437981ce3de9"
						}
					]
				}
			]
		}`,
		resp.Body.String(),
	)
}
