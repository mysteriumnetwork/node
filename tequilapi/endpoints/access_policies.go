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

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model AccessPolicies
type accessPolicyCollection struct {
	Entries []accessPolicy `json:"entries"`
}

type accessPolicy struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Allow       []accessRule `json:"allow"`
}

type accessRule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type accessPoliciesEndpoint struct {
}

var staticAccessPolicy = accessPolicy{
	ID:          "mysterium",
	Title:       "Mysterium verified traffic",
	Description: "Mysterium Network approved identities",
	Allow: []accessRule{
		{
			Type:  "identity",
			Value: "0xf4d6ffba09d460ebe10d24667770437981ce3de9",
		},
	},
}

// NewAccessPoliciesEndpoint creates and returns access policies endpoint
func NewAccessPoliciesEndpoint() *accessPoliciesEndpoint {
	return &accessPoliciesEndpoint{}
}

// swagger:operation GET /access-policies AccessPolicies
// ---
// summary: Returns access lists
// description: Returns list of access policies
// responses:
//   200:
//     description: List of access policies
//     schema:
//       "$ref": "#/definitions/AccessPolicies"
func (ape *accessPoliciesEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	r := accessPolicyCollection{Entries: []accessPolicy{staticAccessPolicy}}
	utils.WriteAsJSON(r, resp)
}

// AddRoutesForAccessPolicies attaches access policies endpoints to router
func AddRoutesForAccessPolicies(router *httprouter.Router) {
	ape := NewAccessPoliciesEndpoint()
	router.GET("/access-policies", ape.List)
}
