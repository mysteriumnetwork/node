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
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vcraescu/go-paginator"
	"github.com/vcraescu/go-paginator/adapter"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchRegistrationFees() (registry.FeesResponse, error)
	FetchSettleFees() (registry.FeesResponse, error)
	FetchStakeDecreaseFee() (registry.FeesResponse, error)
	RegisterIdentity(id string, stake, fee *big.Int, beneficiary string) error
	DecreaseStake(id string, amount, transactorFee uint64) error
}

// promiseSettler settles the given promises
type promiseSettler interface {
	ForceSettle(providerID identity.Identity, hermesID common.Address) error
	SettleWithBeneficiary(id identity.Identity, beneficiary, hermesID common.Address) error
	GetHermesFee(common.Address) (uint16, error)
	SettleIntoStake(providerID identity.Identity, hermesID common.Address) error
}

type settlementHistoryProvider interface {
	List(pingpong.SettlementHistoryFilter) ([]pingpong.SettlementHistoryEntry, error)
}

type transactorEndpoint struct {
	transactor                Transactor
	promiseSettler            promiseSettler
	settlementHistoryProvider settlementHistoryProvider
	hermesAddress             common.Address
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(transactor Transactor, promiseSettler promiseSettler, settlementHistoryProvider settlementHistoryProvider, hermesID common.Address) *transactorEndpoint {
	return &transactorEndpoint{
		transactor:                transactor,
		promiseSettler:            promiseSettler,
		settlementHistoryProvider: settlementHistoryProvider,
		hermesAddress:             hermesID,
	}
}

// swagger:operation GET /transactor/fees FeesDTO
// ---
// summary: Returns fees
// description: Returns fees applied by Transactor
// responses:
//   200:
//     description: fees applied by Transactor
//     schema:
//       "$ref": "#/definitions/FeesDTO"
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
	decreaseStakeFees, err := te.transactor.FetchStakeDecreaseFee()
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	hermesFees, err := te.promiseSettler.GetHermesFee(te.hermesAddress)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	f := contract.FeesDTO{
		Registration:  registrationFees.Fee,
		Settlement:    settlementFees.Fee,
		Hermes:        hermesFees,
		DecreaseStake: decreaseStakeFees.Fee,
	}

	utils.WriteAsJSON(f, resp)
}

// SettleRequest represents the request to settle hermes promises
// swagger:model SettleRequest
type SettleRequest struct {
	HermesID   string `json:"hermes_id"`
	ProviderID string `json:"provider_id"`
}

// swagger:operation POST /transactor/settle/sync SettleSync
// ---
// summary: forces the settlement of promises for the given provider and hermes
// description: Forces a settlement for the hermes promises and blocks until the settlement is complete.
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
// summary: forces the settlement of promises for the given provider and hermes
// description: Forces a settlement for the hermes promises. Does not wait for completion.
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
	err := te.settle(request, func(provider identity.Identity, hermes common.Address) error {
		go func() {
			err := te.promiseSettler.ForceSettle(provider, hermes)
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

	return errors.Wrap(settler(identity.FromAddress(req.ProviderID), common.HexToAddress(req.HermesID)), "settling failed")
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

	req := &contract.IdentityRegistrationRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse identity registration request"), http.StatusBadRequest)
		return
	}

	err = te.transactor.RegisterIdentity(identity, req.Stake, req.Fee, req.Beneficiary)
	if err != nil {
		log.Err(err).Msgf("Failed identity registration request for ID: %s, %+v", identity, req)
		utils.SendError(resp, errors.Wrap(err, "failed identity registration request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (te *transactorEndpoint) SetBeneficiary(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	req := &client.SettleWithBeneficiaryRequest{}
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, fmt.Errorf("failed to parse set beneficiary request: %w", err), http.StatusBadRequest)
		return
	}

	err = te.promiseSettler.SettleWithBeneficiary(identity.FromAddress(id), common.HexToAddress(req.Beneficiary), common.HexToAddress(req.HermesID))
	if err != nil {
		log.Err(err).Msgf("Failed set beneficiary request for ID: %s, %+v", id, req)
		utils.SendError(resp, fmt.Errorf("failed set beneficiary request: %w", err), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation GET /settle/history SettlementHistory
// ---
// summary: Returns settlement history
// description: Returns settlement history
// parameters:
//   - in: query
//     name: date_from
//     description: To filter the settlements from this date. Formatted in RFC3339 e.g. 2020-07-01T00:00:00Z.
//     type: string
//   - in: query
//     name: date_to
//     description: To filter the settlements until this date. Formatted in RFC3339 e.g. 2020-07-01T00:00:00Z.
//     type: string
//   - in: query
//     name: provider_id
//     description: Provider ID to filter the settlements by.
//     type: string
//   - in: query
//     name: hermes_id
//     description: Hermes ID to filter the sessions by.
//     type: string
// responses:
//   200:
//     description: Returns settlement history
//     schema:
//       "$ref": "#/definitions/ListSettlementsResponse"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettlementHistory(resp http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	filter := pingpong.SettlementHistoryFilter{}

	dateFrom := time.Now().AddDate(0, 0, -30)
	if fromStr := req.URL.Query().Get("settled_at_from"); fromStr != "" {
		var err error
		if dateFrom, err = time.Parse(time.RFC3339, fromStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}
	filter.TimeFrom = &dateFrom

	dateTo := time.Now()
	if toStr := req.URL.Query().Get("settled_at_to"); toStr != "" {
		var err error
		if dateTo, err = time.Parse(time.RFC3339, toStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}
	filter.TimeFrom = &dateTo

	if param := req.URL.Query().Get("provider_id"); param != "" {
		providerID := identity.FromAddress(param)
		filter.ProviderID = &providerID
	}
	if param := req.URL.Query().Get("hermes_id"); param != "" {
		hermesID := common.HexToAddress(param)
		filter.HermesID = &hermesID
	}

	page := 1
	if pageStr := req.URL.Query().Get("page"); pageStr != "" {
		var err error
		if page, err = strconv.Atoi(pageStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}

	pageSize := 50
	if pageSizeStr := req.URL.Query().Get("page_size"); pageSizeStr != "" {
		var err error
		if pageSize, err = strconv.Atoi(pageSizeStr); err != nil {
			utils.SendError(resp, err, http.StatusBadRequest)
			return
		}
	}

	settlementsAll, err := te.settlementHistoryProvider.List(filter)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var settlements []pingpong.SettlementHistoryEntry
	p := paginator.New(adapter.NewSliceAdapter(settlementsAll), pageSize)
	p.SetPage(page)
	if err := p.Results(&settlements); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	response := contract.NewSettlementListResponse(settlements, &p)
	utils.WriteAsJSON(response, resp)
}

// DecreaseStakeRequest represents the decrease stake request
// swagger:model DecreaseStakeRequest
type DecreaseStakeRequest struct {
	ID            string `json:"id,omitempty"`
	Amount        uint64 `json:"amount,omitempty"`
	TransactorFee uint64 `json:"transactor_fee,omitempty"`
}

// swagger:operation POST /transactor/stake/decrease Decrease Stake
// ---
// summary: Decreases stake
// description: Decreases stake on eth blockchain via the mysterium transactor.
// parameters:
// - in: body
//   name: body
//   description: decrease stake request
//   schema:
//     $ref: "#/definitions/DecreaseStakeRequest"
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
func (te *transactorEndpoint) DecreaseStake(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	var body DecreaseStakeRequest
	err := json.NewDecoder(request.Body).Decode(&body)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse decrease stake"), http.StatusBadRequest)
		return
	}

	err = te.transactor.DecreaseStake(body.ID, body.Amount, body.TransactorFee)
	if err != nil {
		log.Err(err).Msgf("Failed decreases stake request for ID: %s, %+v", body.ID, body)
		utils.SendError(resp, errors.Wrap(err, "failed decreases stake request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation POST /transactor/stake/increase/sync StakeIncreaseSync
// ---
// summary: forces the settlement with stake increase of promises for the given provider and hermes.
// description: Forces a settlement with stake increase for the hermes promises and blocks until the settlement is complete.
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
func (te *transactorEndpoint) SettleIntoStakeSync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, te.promiseSettler.SettleIntoStake)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// swagger:operation POST /transactor/stake/increase/async StakeIncreaseAsync
// ---
// summary: forces the settlement with stake increase of promises for the given provider and hermes.
// description: Forces a settlement with stake increase for the hermes promises and does not block.
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
func (te *transactorEndpoint) SettleIntoStakeAsync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, func(provider identity.Identity, hermes common.Address) error {
		go func() {
			err := te.promiseSettler.SettleIntoStake(provider, hermes)
			if err != nil {
				log.Error().Err(err).Msgf("could not settle into stake provider(%q) promises", provider.Address)
			}
		}()
		return nil
	})
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

// AddRoutesForTransactor attaches Transactor endpoints to router
func AddRoutesForTransactor(router *httprouter.Router, transactor Transactor, promiseSettler promiseSettler, settlementHistoryProvider settlementHistoryProvider, hermesAddress common.Address) {
	te := NewTransactorEndpoint(transactor, promiseSettler, settlementHistoryProvider, hermesAddress)
	router.POST("/identities/:id/register", te.RegisterIdentity)
	router.POST("/identities/:id/beneficiary", te.SetBeneficiary)
	router.GET("/transactor/fees", te.TransactorFees)
	router.POST("/transactor/settle/sync", te.SettleSync)
	router.POST("/transactor/settle/async", te.SettleAsync)
	router.GET("/transactor/settle/history", te.SettlementHistory)
	router.POST("/transactor/stake/increase/sync", te.SettleIntoStakeSync)
	router.POST("/transactor/stake/increase/async", te.SettleIntoStakeAsync)
	router.POST("/transactor/stake/decrease", te.DecreaseStake)
}
