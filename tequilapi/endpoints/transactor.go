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

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchRegistrationFees() (registry.FeesResponse, error)
	FetchSettleFees() (registry.FeesResponse, error)
	TopUp(identity string) error
	RegisterIdentity(identity string, regReqDTO *registry.IdentityRegistrationRequestDTO) error
}

// promiseSettler settles the given promises
type promiseSettler interface {
	ForceSettle(providerID identity.Identity, accountantID common.Address) error
}

type transactorEndpoint struct {
	transactor     Transactor
	promiseSettler promiseSettler
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(transactor Transactor, promiseSettler promiseSettler) *transactorEndpoint {
	return &transactorEndpoint{
		transactor:     transactor,
		promiseSettler: promiseSettler,
	}
}

// Fees represents the transactor fees
// swagger:model Fees
type Fees struct {
	Registration uint64 `json:"registration"`
	Settlement   uint64 `json:"settlement"`
}

// swagger:operation GET /transactor/fees Fees
// ---
// summary: Returns fees
// description: Returns fees applied by Transactor
// responses:
//   200:
//     description: fees applied by Transactor
//     schema:
//       "$ref": "#/definitions/Fees"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) TransactorFees(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	registrationFees, err := te.transactor.FetchRegistrationFees()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	settlementFees, err := te.transactor.FetchSettleFees()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	f := Fees{
		Registration: registrationFees.Fee,
		Settlement:   settlementFees.Fee,
	}

	utils.WriteAsJSON(f, resp)
}

// SettleRequest represents the request to settle accountant promises
// swagger:model SettleRequest
type SettleRequest struct {
	AccountantID string `json:"accountant_id"`
	ProviderID   string `json:"provider_id"`
}

// swagger:operation POST /transactor/settle/sync SettleSync
// ---
// summary: forces the settlement of promises for the given provider and accountant
// description: Forces a settlement for the accountant promises and blocks until the settlement is complete.
// parameters:
// - in: body
//   name: body
//   description: settle request body
//   schema:
//     $ref: "#/definitions/SettleRequest"
// responses:
//   202:
//     description: settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettleSync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, te.promiseSettler.ForceSettle)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// swagger:operation POST /transactor/settle/async SettleAsync
// ---
// summary: forces the settlement of promises for the given provider and accountant
// description: Forces a settlement for the accountant promises. Does not wait for completion.
// parameters:
// - in: body
//   name: body
//   description: settle request body
//   schema:
//     $ref: "#/definitions/SettleRequest"
// responses:
//   202:
//     description: settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettleAsync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, func(provider identity.Identity, accountant common.Address) error {
		go func() {
			err := te.promiseSettler.ForceSettle(provider, accountant)
			if err != nil {
				log.Error().Err(err).Msgf("could not settle provider(%q) promises", provider.Address)
			}
		}()
		return nil
	})
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (te *transactorEndpoint) settle(request *http.Request, settler func(identity.Identity, common.Address) error) error {
	req := SettleRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal settle request")
	}

	return errors.Wrap(settler(identity.FromAddress(req.ProviderID), common.HexToAddress(req.AccountantID)), "settling failed")
}

// swagger:operation POST /transactor/topup
// ---
// summary: tops up myst to the given identity
// description: tops up myst to the given identity
// parameters:
// - in: body
//   name: body
//   description: top up request body
//   schema:
//     $ref: "#/definitions/TopUpRequestDTO"
// responses:
//   202:
//     description: top up request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) TopUp(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	topUpDTO := registry.TopUpRequest{}

	err := json.NewDecoder(request.Body).Decode(&topUpDTO)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse top up request"), http.StatusBadRequest)
		return
	}

	err = te.transactor.TopUp(topUpDTO.Identity)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation POST /identities/{id}/register Identity RegisterIdentity
// ---
// summary: Registers identity
// description: Registers identity on Mysterium Network smart contracts using Transactor
// parameters:
// - name: id
//   in: path
//   description: Identity address to register
//   type: string
//   required: true
// - in: body
//   name: body
//   description: all body parameters a optional
//   schema:
//     $ref: "#/definitions/IdentityRegistrationRequestDTO"
// responses:
//   200:
//     description: Payout info registered
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) RegisterIdentity(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	identity := params.ByName("id")

	regReqDTO := &registry.IdentityRegistrationRequestDTO{}

	err := json.NewDecoder(request.Body).Decode(&regReqDTO)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse identity registration request"), http.StatusBadRequest)
		return
	}

	err = te.transactor.RegisterIdentity(identity, regReqDTO)
	if err != nil {
		log.Err(err).Msgf("Failed identity registration request for ID: %s, %+v", identity, regReqDTO)
		utils.SendError(resp, errors.Wrap(err, "failed identity registration request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// AddRoutesForTransactor attaches Transactor endpoints to router
func AddRoutesForTransactor(router *httprouter.Router, transactor Transactor, promiseSettler promiseSettler) {
	te := NewTransactorEndpoint(transactor, promiseSettler)
	router.POST("/identities/:id/register", te.RegisterIdentity)
	router.GET("/transactor/fees", te.TransactorFees)
	router.POST("/transactor/topup", te.TopUp)
	router.POST("/transactor/settle/sync", te.SettleSync)
	router.POST("/transactor/settle/async", te.SettleAsync)
}
