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
	"github.com/mysteriumnetwork/node/requests"
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
	http                    *requests.HTTPClient
	accessPolicyEndpointURL string
}

// NewAccessPoliciesEndpoint creates and returns access policies endpoint
func NewAccessPoliciesEndpoint(httpClient *requests.HTTPClient, accessPolicyEndpointURL string) *accessPoliciesEndpoint {
	return &accessPoliciesEndpoint{
		http:                    httpClient,
		accessPolicyEndpointURL: accessPolicyEndpointURL,
	}
}

// swagger:operation GET /access-policies AccessPolicies
// ---
// summary: Returns access policies
// description: Returns list of access policies
// responses:
//   200:
//     description: List of access policies
//     schema:
//       "$ref": "#/definitions/AccessPolicies"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ape *accessPoliciesEndpoint) List(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	req, err := requests.NewGetRequest(ape.accessPolicyEndpointURL, "", nil)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	r := accessPolicyCollection{}
	err = ape.http.DoRequestAndParseResponse(req, &r)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(r, resp)
}

// AddRoutesForAccessPolicies attaches access policies endpoints to router
func AddRoutesForAccessPolicies(httpClient *requests.HTTPClient, router *httprouter.Router, accessPolicyEndpointURL string) {
	ape := NewAccessPoliciesEndpoint(httpClient, accessPolicyEndpointURL)
	router.GET("/access-policies", ape.List)
}
