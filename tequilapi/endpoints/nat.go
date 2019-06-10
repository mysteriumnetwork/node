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
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type stateProvider func() stateEvent.State

// NATEndpoint struct represents endpoints about NAT traversal
type NATEndpoint struct {
	stateProvider stateProvider
}

// NewNATEndpoint creates and returns nat endpoint
func NewNATEndpoint(stateProvider stateProvider) *NATEndpoint {
	return &NATEndpoint{
		stateProvider: stateProvider,
	}
}

// NATStatus provides NAT configuration info
// swagger:operation GET /nat/status NAT NATStatusDTO
// ---
// summary: Shows NAT status
// description: NAT status returns the last known NAT traversal status
// responses:
//   200:
//     description: NAT status ("not_finished"/"successful"/"failed") and optionally error if status is "failed"
//     schema:
//       "$ref": "#/definitions/NATStatusDTO"
func (ne *NATEndpoint) NATStatus(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	utils.WriteAsJSON(ne.stateProvider().NATStatus, resp)
}

// AddRoutesForNAT adds nat routes to given router
func AddRoutesForNAT(router *httprouter.Router, stateProvider stateProvider) {
	natEndpoint := NewNATEndpoint(stateProvider)

	router.GET("/nat/status", natEndpoint.NATStatus)
}
