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
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/payout"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpong_event "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/pkg/errors"
)

type balanceProvider interface {
	GetBalance(chainID int64, id identity.Identity) *big.Int
}

type earningsProvider interface {
	GetEarnings(chainID int64, id identity.Identity) pingpong_event.Earnings
}

type beneficiaryProvider interface {
	GetBeneficiary(identity common.Address) (common.Address, error)
}

type providerChannel interface {
	GetProviderChannel(chainID int64, hermesAddress common.Address, provider common.Address, pending bool) (client.ProviderChannel, error)
}

type identityMover interface {
	Import(blob []byte, currPass, newPass string) (identity.Identity, error)
}

type identitiesAPI struct {
	mover             identityMover
	idm               identity.Manager
	selector          identity_selector.Handler
	registry          registry.IdentityRegistry
	channelCalculator *pingpong.AddressProvider
	balanceProvider   balanceProvider
	earningsProvider  earningsProvider
	bc                providerChannel
	transactor        Transactor
	bprovider         beneficiaryProvider
	addressStorage    *payout.AddressStorage
}

// swagger:operation GET /identities Identity listIdentities
// ---
// summary: Returns identities
// description: Returns list of identities
// responses:
//   200:
//     description: List of identities
//     schema:
//       "$ref": "#/definitions/ListIdentitiesResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) List(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	ids := ia.idm.GetIdentities()
	idsDTO := contract.NewIdentityListResponse(ids)
	utils.WriteAsJSON(idsDTO, resp)
}

// swagger:operation PUT /identities/current Identity currentIdentity
// ---
// summary: Returns my current identity
// description: Tries to retrieve the last used identity, the first identity, or creates and returns a new identity
// parameters:
//   - in: body
//     name: body
//     description: Parameter in body (passphrase) required for creating new identity
//     schema:
//       $ref: "#/definitions/IdentityCurrentRequestDTO"
// responses:
//   200:
//     description: Unlocked identity returned
//     schema:
//       "$ref": "#/definitions/IdentityRefDTO"
//   400:
//     description: Bad Request
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
func (ia *identitiesAPI) Current(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	var req contract.IdentityCurrentRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	if errorMap := req.Validate(); errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	idAddress := ""
	if req.Address != nil {
		idAddress = *req.Address
	}

	chainID := config.GetInt64(config.FlagChainID)
	id, err := ia.selector.UseOrCreate(idAddress, *req.Passphrase, chainID)

	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, resp)
}

// swagger:operation POST /identities Identity createIdentity
// ---
// summary: Creates new identity
// description: Creates identity and stores in keystore encrypted with passphrase
// parameters:
//   - in: body
//     name: body
//     description: Parameter in body (passphrase) required for creating new identity
//     schema:
//       $ref: "#/definitions/IdentityCreateRequestDTO"
// responses:
//   200:
//     description: Identity created
//     schema:
//       "$ref": "#/definitions/IdentityRefDTO"
//   400:
//     description: Bad Request
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
func (ia *identitiesAPI) Create(resp http.ResponseWriter, httpReq *http.Request, _ httprouter.Params) {
	var req contract.IdentityCreateRequest
	err := json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	if errorMap := req.Validate(); errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	id, err := ia.idm.CreateNewIdentity(*req.Passphrase)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, resp)
}

// swagger:operation PUT /identities/{id}/unlock Identity unlockIdentity
// ---
// summary: Unlocks identity
// description: Uses passphrase to decrypt identity stored in keystore
// parameters:
// - in: path
//   name: id
//   description: Identity stored in keystore
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Parameter in body (passphrase) required for unlocking identity
//   schema:
//     $ref: "#/definitions/IdentityUnlockRequestDTO"
// responses:
//   202:
//     description: Identity unlocked
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   403:
//     description: Forbidden
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) Unlock(resp http.ResponseWriter, httpReq *http.Request, params httprouter.Params) {
	address := params.ByName("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		utils.SendError(resp, err, http.StatusNotFound)
		return
	}

	var req contract.IdentityUnlockRequest
	err = json.NewDecoder(httpReq.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	if errorMap := req.Validate(); errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	err = ia.idm.Unlock(chainID, id.Address, *req.Passphrase)
	if err != nil {
		utils.SendError(resp, err, http.StatusForbidden)
		return
	}
	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation GET /identities/{id} Identity getIdentity
// ---
// summary: Get identity
// description: Provide identity details
// parameters:
//   - in: path
//     name: id
//     description: hex address of identity
//     type: string
//     required: true
// responses:
//   200:
//     description: Identity retrieved
//     schema:
//       "$ref": "#/definitions/IdentityRefDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) Get(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	address := params.ByName("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		utils.SendError(resp, err, http.StatusNotFound)
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	regStatus, err := ia.registry.GetRegistrationStatus(chainID, id)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to check identity registration status"), http.StatusInternalServerError)
		return
	}

	channelAddress, err := ia.channelCalculator.GetChannelAddress(chainID, id)
	if err != nil {
		utils.SendError(resp, fmt.Errorf("failed to calculate channel address %w", err), http.StatusInternalServerError)
		return
	}

	var stake = new(big.Int)
	if regStatus == registry.Registered {
		hermesID, err := ia.channelCalculator.GetActiveHermes(chainID)
		if err != nil {
			utils.SendError(resp, fmt.Errorf("could not get active hermes %w", err), http.StatusInternalServerError)
			return
		}

		data, err := ia.bc.GetProviderChannel(chainID, hermesID, common.HexToAddress(address), false)
		if err != nil {
			utils.SendError(resp, fmt.Errorf("failed to check identity registration status: %w", err), http.StatusInternalServerError)
			return
		}
		stake = data.Stake
	}

	balance := ia.balanceProvider.GetBalance(chainID, id)
	settlement := ia.earningsProvider.GetEarnings(chainID, id)
	status := contract.IdentityDTO{
		Address:            address,
		RegistrationStatus: regStatus.String(),
		ChannelAddress:     channelAddress.Hex(),
		Balance:            balance,
		Earnings:           settlement.UnsettledBalance,
		EarningsTotal:      settlement.LifetimeBalance,
		Stake:              stake,
	}
	utils.WriteAsJSON(status, resp)
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
//     description: Status retrieved
//     schema:
//       "$ref": "#/definitions/IdentityRegistrationResponseDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) RegistrationStatus(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	address := params.ByName("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		utils.SendError(resp, err, http.StatusNotFound)
		return
	}

	regStatus, err := ia.registry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), id)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to check identity registration status"), http.StatusInternalServerError)
		return
	}

	registrationDataDTO := &contract.IdentityRegistrationResponse{
		Status:     regStatus.String(),
		Registered: regStatus.Registered(),
	}
	utils.WriteAsJSON(registrationDataDTO, resp)
}

// swagger:operation GET /identities/{id}/beneficiary Identity beneficiary address
// ---
// summary: Provide identity beneficiary address
// description: Provides beneficiary address for given identity
// parameters:
//   - in: path
//     name: id
//     description: hex address of identity
//     type: string
//     required: true
// responses:
//   200:
//     description: Beneficiary retrieved
//     schema:
//       "$ref": "#/definitions/IdentityBeneficiaryResponseDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) Beneficiary(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	address := params.ByName("id")
	data, err := ia.bprovider.GetBeneficiary(common.HexToAddress(address))
	if err != nil {
		utils.SendError(resp, fmt.Errorf("failed to check identity registration status: %w", err), http.StatusInternalServerError)
		return
	}

	registrationDataDTO := &contract.IdentityBeneficiaryResponse{
		Beneficiary: data.Hex(),
	}
	utils.WriteAsJSON(registrationDataDTO, resp)
}

// swagger:operation GET /identities/{id}/referral Referral
// ---
// summary: Gets referral token
// description: Gets a referral token for the given identity if a campaign exists
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get a token
//   type: string
//   required: true
// responses:
//   200:
//     description: Token response
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) GetReferralToken(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	tkn, err := ia.transactor.GetReferralToken(common.HexToAddress(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	utils.WriteAsJSON(contract.ReferralTokenResponse{
		Token: tkn,
	}, resp)
}

// swagger:operation GET /identities/{id}/referral-available Referral availability check
// ---
// summary: Checks if the user can obtain a referral token
// description: Verifies user's eligibility and the presence of an applicable public campaign
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get a token
//   type: string
//   required: true
// responses:
//   200:
//     description: Success
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) ReferralTokenAvailable(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	err := ia.transactor.ReferralTokenAvailable(common.HexToAddress(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
}

// swagger:operation POST /identities-import Identities importIdentity
// ---
// summary: Imports a given identity.
// description: Imports a given identity returning it is a blob of text which can later be used to import it back.
// parameters:
// - in: body
//   name: body
//   description: Parameter in body used to import an identity.
//   schema:
//     $ref: "#/definitions/IdentityImportRequest"
// responses:
//   200:
//     description: Unlocked identity returned
//     schema:
//       "$ref": "#/definitions/IdentityRefDTO"
//   400:
//     description: Bad Request error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) Import(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var req contract.IdentityImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, err, http.StatusBadRequest)
		return
	}

	id, err := ia.mover.Import(req.Data, req.CurrentPassphrase, req.NewPassphrase)
	if err != nil {
		utils.SendError(w, fmt.Errorf("failed to import identity: %w", err), http.StatusBadRequest)
		return
	}

	if req.SetDefault {
		if err := ia.selector.SetDefault(id.Address); err != nil {
			utils.SendError(w, fmt.Errorf("failed to set default identity: %w", err), http.StatusBadRequest)
			return
		}
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, w)
}

// swagger:operation GET /identities/:id/payout-address
// ---
// summary: Get payout address
// description: Get payout address stored locally
// parameters:
// - in: path
//   name: id
//   description: Identity stored in keystore
//   type: string
//   required: true
// responses:
//   200:
//     description: Unlocked identity returned
//     schema:
//       "$ref": "#/definitions/PayoutAddressRequest"
//   400:
//     description: Bad Request error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   404:
//     description: Not Found
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) GetPayoutAddress(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	addr, err := ia.addressStorage.Address(id)
	if err != nil {
		if errors.Is(err, payout.ErrNotFound) {
			utils.SendError(w, err, http.StatusNotFound)
			return
		}
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.PayoutAddressRequest{Address: addr}, w)
}

// swagger:operation PUT /identities/:id/payout-address
// ---
// summary: Save payout address
// description: Stores payout address locally
// parameters:
// - in: path
//   name: id
//   description: Identity stored in keystore
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Payout address request.
//   schema:
//     $ref: "#/definitions/PayoutAddressRequest"
// responses:
//   200:
//     description: Unlocked identity returned
//     schema:
//       "$ref": "#/definitions/PayoutAddressRequest"
//   400:
//     description: Bad Request error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ia *identitiesAPI) SavePayoutAddress(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	var par contract.PayoutAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&par); err != nil {
		utils.SendError(w, err, http.StatusBadRequest)
		return
	}

	err := ia.addressStorage.Save(id, par.Address)
	if err != nil {
		utils.SendError(w, err, http.StatusBadRequest)
		return
	}

	utils.WriteAsJSON(par, w)
}

// AddRoutesForIdentities creates /identities endpoint on tequilapi service
func AddRoutesForIdentities(
	router *httprouter.Router,
	idm identity.Manager,
	selector identity_selector.Handler,
	registry registry.IdentityRegistry,
	balanceProvider balanceProvider,
	channelAddressCalculator *pingpong.AddressProvider,
	earningsProvider earningsProvider,
	bc providerChannel,
	transactor Transactor,
	bprovider beneficiaryProvider,
	mover identityMover,
	addressStorage *payout.AddressStorage,
) {
	idAPI := &identitiesAPI{
		mover:             mover,
		idm:               idm,
		selector:          selector,
		registry:          registry,
		balanceProvider:   balanceProvider,
		channelCalculator: channelAddressCalculator,
		earningsProvider:  earningsProvider,
		bc:                bc,
		transactor:        transactor,
		bprovider:         bprovider,
		addressStorage:    addressStorage,
	}
	router.GET("/identities", idAPI.List)
	router.POST("/identities", idAPI.Create)
	router.PUT("/identities/:id", func(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
		// TODO: remove this hack when we replace our router
		switch params.ByName("id") {
		case "current":
			idAPI.Current(resp, request, params)
		default:
			http.NotFound(resp, request)
		}
	})
	router.GET("/identities/:id", idAPI.Get)
	router.GET("/identities/:id/status", idAPI.Get)
	router.PUT("/identities/:id/unlock", idAPI.Unlock)
	router.GET("/identities/:id/registration", idAPI.RegistrationStatus)
	router.GET("/identities/:id/beneficiary", idAPI.Beneficiary)
	router.GET("/identities/:id/referral", idAPI.GetReferralToken)
	router.GET("/identities/:id/referral-available", idAPI.ReferralTokenAvailable)
	router.GET("/identities/:id/payout-address", idAPI.GetPayoutAddress)
	router.PUT("/identities/:id/payout-address", idAPI.SavePayoutAddress)

	router.POST("/identities-import", idAPI.Import)
}
