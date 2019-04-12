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

// swagger:model ACL
type acl struct {
	AccessLists []accessList `json:"acl"`
}

type accessList struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Allow       []accessRule `json:"allow"`
}

type accessRule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type aclEndpoint struct {
}

var staticAccessList = accessList{
	Name:        "mysterium",
	Description: "Mysterium Network approved identities",
	Allow: []accessRule{
		{
			Type:  "identity",
			Value: "0xf4d6ffba09d460ebe10d24667770437981ce3de9",
		},
	},
}

// NewACLEndpoint creates and returns ACL endpoint
func NewACLEndpoint() *aclEndpoint {
	return &aclEndpoint{}
}

// swagger:operation GET /acl ACL
// ---
// summary: Returns access lists
// description: Returns list of access lists
// responses:
//   200:
//     description: List of access lists
//     schema:
//       "$ref": "#/definitions/ACL"
func (ale *aclEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	r := acl{AccessLists: []accessList{staticAccessList}}
	utils.WriteAsJSON(r, resp)
}

// AddRoutesForACL attaches ACL endpoints to router
func AddRoutesForACL(router *httprouter.Router) {
	ale := NewACLEndpoint()
	router.GET("/acl", ale.List)
}
