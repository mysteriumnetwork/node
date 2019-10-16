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
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/core/transactor"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchFees() (transactor.Fees, error)
	TopUp(identity string) error
	RegisterIdentity(identity string, regReqDTO *transactor.IdentityRegistrationRequestDTO) error
}

type transactorEndpoint struct {
	transactor Transactor
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(transactor Transactor) *transactorEndpoint {
	return &transactorEndpoint{
		transactor: transactor,
	}
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
	fees, err := te.transactor.FetchFees()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	utils.WriteAsJSON(fees, resp)
}

// swagger:operation POST /transactor/topup ErrorMessageDTO
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
	topUpDTO := &transactor.TopUpRequest{}

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

	regReqDTO := &transactor.IdentityRegistrationRequestDTO{}

	err := json.NewDecoder(request.Body).Decode(&regReqDTO)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse identity registration request"), http.StatusBadRequest)
		return
	}

	err = te.transactor.RegisterIdentity(identity, regReqDTO)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed identity registration request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// AddRoutesForTransactor attaches Transactor endpoints to router
func AddRoutesForTransactor(router *httprouter.Router, transactor Transactor) {
	te := NewTransactorEndpoint(transactor)
	router.POST("/identities/:id/register", te.RegisterIdentity)
	router.GET("/transactor/fees", te.TransactorFees)
	router.POST("/transactor/topup", te.TopUp)
}
