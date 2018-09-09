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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// SignatureDTO represents Elliptic Curve signature parts
//
// swagger:model DecomposedSignatureDTO
type SignatureDTO struct {
	// S part of signature
	// example: "0x1321313212312..."
	R string `json:"r"`
	// R part of signature
	// example: "0x1234563564354..."
	S string `json:"s"`
	// Sign - 27 or 28 as expected by ethereum ecrecover function
	// example: 27
	V uint8 `json:"v"`
}

// PublicKeyPartsDTO represents ECDSA public key with first byte stripped (0x04) and splitted into two 32 bytes size arrays
//
// swagger:model PublicKeyPartsDTO
type PublicKeyPartsDTO struct {
	// First 32 bytes of public key in hex representation
	// example: "0x1321313212312..."
	Part1 string `json:"part1"`
	// Last 32 bytes of public key inx hex representation
	// example: "0x1321313212312..."
	Part2 string `json:"part2"`
}

// RegistrationDataDTO represents registration status and needed data for registering of given identity
//
// swagger:model RegistrationDataDTO
type RegistrationDataDTO struct {
	// Returns true if identity is registered in payments smart contract
	Registered bool `json:"registered"`

	PublicKey PublicKeyPartsDTO `json:"publicKey"`

	Signature SignatureDTO `json:"signature"`
}

type registrationEndpoint struct {
	dataProvider   RegistrationDataProvider
	statusProvider IdentityRegistry
}

func newRegistrationEndpoint(dataProvider RegistrationDataProvider, statusProvider IdentityRegistry) *registrationEndpoint {
	return &registrationEndpoint{
		dataProvider:   dataProvider,
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
//     example: "0x0000000000000000000000000000000000000001"
//     type: string
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

	registrationData, err := endpoint.dataProvider.ProvideRegistrationData(id)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationDataDTO := &RegistrationDataDTO{
		Registered: isRegistered,
		PublicKey: PublicKeyPartsDTO{
			Part1: hexutil.Encode(registrationData.PublicKey.Part1),
			Part2: hexutil.Encode(registrationData.PublicKey.Part2),
		},
		Signature: SignatureDTO{
			R: hexutil.Encode(registrationData.Signature.R[:]),
			S: hexutil.Encode(registrationData.Signature.S[:]),
			V: registrationData.Signature.V,
		},
	}
	utils.WriteAsJSON(registrationDataDTO, resp)
}

// AddIdentityRegistrationEndpoint adds identity registration data endpoint to given http router
func AddIdentityRegistrationEndpoint(router *httprouter.Router, dataProvider RegistrationDataProvider, statusProvider IdentityRegistry) {

	registrationEndpoint := newRegistrationEndpoint(
		dataProvider,
		statusProvider,
	)

	router.GET("/identities/:id/registration", registrationEndpoint.IdentityRegistrationData)
}
