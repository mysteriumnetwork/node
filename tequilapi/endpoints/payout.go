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

package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// swagger:model PayoutInfoDTO
type payoutInfo struct {
	// in Ethereum address format
	// required: true
	// example: 0x000000000000000000000000000000000000000a
	EthAddress *string `json:"ethAddress"`
}

// PayoutInfoRegistry allows to register payout info
type PayoutInfoRegistry interface {
	UpdatePayoutInfo(id identity.Identity, ethAddress string, signer identity.Signer) error
}

type payoutAPI struct {
	idm                identity.Manager
	signerFactory      identity.SignerFactory
	payoutInfoRegistry PayoutInfoRegistry
}

// NewPayoutEndpoint creates payout api controller used by tequilapi service
func NewPayoutEndpoint(idm identity.Manager, signerFactory identity.SignerFactory, payoutInfoRegistry PayoutInfoRegistry) *payoutAPI {
	return &payoutAPI{idm, signerFactory, payoutInfoRegistry}
}

// swagger:operation PUT /identities/{id}/payout Identity updatePayoutInfo
// ---
// summary: Registers payout info
// description: Registers payout address for identity
// parameters:
// - name: id
//   in: path
//   description: Identity stored in keystore
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Parameter in body (ethAddress) is required
//   schema:
//     $ref: "#/definitions/PayoutInfoDTO"
// responses:
//   200:
//     description: Payout info registered
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *payoutAPI) UpdatePayoutInfo(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := identity.FromAddress(params.ByName("id"))
	payoutInfoReq, err := toPayoutInfoRequest(request)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	errorMap := validatePayoutInfoRequest(payoutInfoReq)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	err = endpoint.payoutInfoRegistry.UpdatePayoutInfo(id, *payoutInfoReq.EthAddress, endpoint.signerFactory(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

func toPayoutInfoRequest(req *http.Request) (*payoutInfo, error) {
	var payoutReq = &payoutInfo{}
	err := json.NewDecoder(req.Body).Decode(&payoutReq)
	return payoutReq, err
}

func validatePayoutInfoRequest(req *payoutInfo) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	if req.EthAddress == nil {
		errors.ForField("eth_address").AddError("required", "Field is required")
	}
	// TODO: implement validation of eth address
	return
}

// AddRoutesForPayout creates payout endpoint on tequilapi service
func AddRoutesForPayout(
	router *httprouter.Router,
	idm identity.Manager,
	signerFactory identity.SignerFactory,
	payoutInfoRegistry PayoutInfoRegistry,
) {
	idmEnd := NewPayoutEndpoint(idm, signerFactory, payoutInfoRegistry)
	router.PUT("/identities/:id/payout", idmEnd.UpdatePayoutInfo)
}
