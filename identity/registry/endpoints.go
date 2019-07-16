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

package registry

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// RegistrationDataDTO represents registration status and needed data for registering of given identity
//
// swagger:model RegistrationDataDTO
type RegistrationDataDTO struct {
	// Returns true if identity is registered in payments smart contract
	Registered bool `json:"registered"`
}

type registrationEndpoint struct {
	statusProvider IdentityRegistry
}

func newRegistrationEndpoint(statusProvider IdentityRegistry) *registrationEndpoint {
	return &registrationEndpoint{
		statusProvider: statusProvider,
	}
}

// swagger:operation GET /identities/{id}/registration Identity identityRegistration
// ---
// summary: Provide identity registration status
// description: Provides registration status for given identity, if identity is not registered - provides additional data required for identity registration
// parameters:
//   - in: path
//     name: id
//     description: hex address of identity
//     type: string
//     required: true
// responses:
//   200:
//     description: Registration status and data
//     schema:
//       "$ref": "#/definitions/RegistrationDataDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *registrationEndpoint) IdentityRegistrationData(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := identity.FromAddress(params.ByName("id"))

	isRegistered, err := endpoint.statusProvider.IsRegistered(id)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationDataDTO := &RegistrationDataDTO{
		Registered: isRegistered,
	}
	utils.WriteAsJSON(registrationDataDTO, resp)
}

// AddIdentityRegistrationEndpoint adds identity registration data endpoint to given http router
func AddIdentityRegistrationEndpoint(router *httprouter.Router, statusProvider IdentityRegistry) {

	registrationEndpoint := newRegistrationEndpoint(
		statusProvider,
	)

	router.GET("/identities/:id/registration", registrationEndpoint.IdentityRegistrationData)
}
