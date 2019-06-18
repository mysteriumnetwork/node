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
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// swagger:model PayoutInfoDTO
type payoutInfo struct {
	// in Ethereum address format
	// required: true
	// example: 0x000000000000000000000000000000000000000a
	EthAddress string `json:"ethAddress"`
}

// swagger:model ReferralInfoDTO
type referralInfo struct {
	// required: true
	// example: ABC123
	ReferralCode string `json:"referralCode"`
}

type payoutInfoResponse struct {
	EthAddress   string `json:"ethAddress"`
	ReferralCode string `json:"referralCode"`
}

// PayoutInfoRegistry allows to register payout info
type PayoutInfoRegistry interface {
	GetPayoutInfo(id identity.Identity, signer identity.Signer) (*mysterium.PayoutInfoResponse, error)
	UpdatePayoutInfo(id identity.Identity, ethAddress string, signer identity.Signer) error
	UpdateReferralInfo(id identity.Identity, referralCode string, signer identity.Signer) error
}

type payoutEndpoint struct {
	idm                identity.Manager
	signerFactory      identity.SignerFactory
	payoutInfoRegistry PayoutInfoRegistry
}

// NewPayoutEndpoint creates payout api endpoint
func NewPayoutEndpoint(idm identity.Manager, signerFactory identity.SignerFactory, payoutInfoRegistry PayoutInfoRegistry) *payoutEndpoint {
	return &payoutEndpoint{idm, signerFactory, payoutInfoRegistry}
}

func (endpoint *payoutEndpoint) GetPayoutInfo(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := identity.FromAddress(params.ByName("id"))
	payoutInfo, err := endpoint.payoutInfoRegistry.GetPayoutInfo(id, endpoint.signerFactory(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusNotFound)
		return
	}

	response := &payoutInfoResponse{
		EthAddress:   payoutInfo.EthAddress,
		ReferralCode: payoutInfo.ReferralCode,
	}
	utils.WriteAsJSON(response, resp)
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
func (endpoint *payoutEndpoint) UpdatePayoutInfo(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

	err = endpoint.payoutInfoRegistry.UpdatePayoutInfo(
		id,
		payoutInfoReq.EthAddress,
		endpoint.signerFactory(id),
	)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// swagger:operation PUT /identities/{id}/referral Identity updateReferralInfo
// ---
// summary: Registers referral info
// description: Registers referral code for identity
// parameters:
// - name: id
//   in: path
//   description: Identity stored in keystore
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Parameter in body (referral_code) is required
//   schema:
//     $ref: "#/definitions/ReferralInfoDTO"
// responses:
//   200:
//     description: Referral info registered
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *payoutEndpoint) UpdateReferralInfo(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := identity.FromAddress(params.ByName("id"))

	referralInfoReq, err := toReferralInfoRequest(request)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	err = endpoint.payoutInfoRegistry.UpdateReferralInfo(
		id,
		referralInfoReq.ReferralCode,
		endpoint.signerFactory(id),
	)
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

func toReferralInfoRequest(req *http.Request) (*referralInfo, error) {
	var referralReq = &referralInfo{}
	err := json.NewDecoder(req.Body).Decode(&referralReq)
	return referralReq, err
}

func validatePayoutInfoRequest(req *payoutInfo) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	if req.EthAddress == "" {
		errors.ForField("ethAddress").AddError("required", "Field is required")
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
	router.GET("/identities/:id/payout", idmEnd.GetPayoutInfo)
	router.PUT("/identities/:id/payout", idmEnd.UpdatePayoutInfo)
	router.PUT("/identities/:id/referral", idmEnd.UpdateReferralInfo)
}
