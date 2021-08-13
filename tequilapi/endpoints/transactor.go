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
	errs "errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/asdine/storm/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vcraescu/go-paginator/adapter"
)

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchRegistrationFees(chainID int64) (registry.FeesResponse, error)
	FetchSettleFees(chainID int64) (registry.FeesResponse, error)
	FetchStakeDecreaseFee(chainID int64) (registry.FeesResponse, error)
	RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error
	DecreaseStake(id string, chainID int64, amount, transactorFee *big.Int) error
	GetTokenReward(referralToken string) (registry.TokenRewardResponse, error)
	GetReferralToken(id common.Address) (string, error)
	ReferralTokenAvailable(id common.Address) error
}

// promiseSettler settles the given promises
type promiseSettler interface {
	ForceSettle(chainID int64, providerID identity.Identity, hermesID common.Address) error
	GetHermesFee(chainID int64, id common.Address) (uint16, error)
	SettleIntoStake(chainID int64, providerID identity.Identity, hermesID common.Address) error
	Withdraw(chainID int64, providerID identity.Identity, hermesID, beneficiary common.Address) error
}

type addressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
}

type beneficiarySaver interface {
	SettleAndSaveBeneficiary(id identity.Identity, beneficiary common.Address) error
	CleanupAndGetChangeStatus(id identity.Identity, currentBeneficiary string) (*beneficiary.ChangeStatus, error)
}

type settlementHistoryProvider interface {
	List(pingpong.SettlementHistoryFilter) ([]pingpong.SettlementHistoryEntry, error)
}

type transactorEndpoint struct {
	transactor                Transactor
	identityRegistry          identityRegistry
	promiseSettler            promiseSettler
	settlementHistoryProvider settlementHistoryProvider
	addressProvider           addressProvider
	bhandler                  beneficiarySaver
	bprovider                 beneficiaryProvider
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(
	transactor Transactor,
	identityRegistry identityRegistry,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
	bhander beneficiarySaver,
	bprovider beneficiaryProvider,
) *transactorEndpoint {
	return &transactorEndpoint{
		transactor:                transactor,
		identityRegistry:          identityRegistry,
		promiseSettler:            promiseSettler,
		settlementHistoryProvider: settlementHistoryProvider,
		addressProvider:           addressProvider,
		bhandler:                  bhander,
		bprovider:                 bprovider,
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
	chainID := config.GetInt64(config.FlagChainID)
	registrationFees, err := te.transactor.FetchRegistrationFees(chainID)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	settlementFees, err := te.transactor.FetchSettleFees(chainID)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	decreaseStakeFees, err := te.transactor.FetchStakeDecreaseFee(chainID)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	hermes, err := te.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	hermesFees, err := te.promiseSettler.GetHermesFee(chainID, hermes)
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

// swagger:operation POST /transactor/settle/sync SettleSync
// ---
// summary: forces the settlement of promises for the given provider and hermes
// description: Forces a settlement for the hermes promises and blocks until the settlement is complete.
// parameters:
// - in: body
//   name: body
//   description: settle request body
//   schema:
//     $ref: "#/definitions/SettleRequestDTO"
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
//     $ref: "#/definitions/SettleRequestDTO"
// responses:
//   202:
//     description: settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettleAsync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, func(chainID int64, provider identity.Identity, hermes common.Address) error {
		go func() {
			err := te.promiseSettler.ForceSettle(chainID, provider, hermes)
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

func (te *transactorEndpoint) settle(request *http.Request, settler func(int64, identity.Identity, common.Address) error) error {
	req := contract.SettleRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal settle request")
	}

	chainID := config.GetInt64(config.FlagChainID)
	return errors.Wrap(settler(chainID, identity.FromAddress(req.ProviderID), common.HexToAddress(req.HermesID)), "settling failed")
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
//     $ref: "#/definitions/IdentityRegisterRequestDTO"
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
	id := identity.FromAddress(params.ByName("id"))
	chainID := config.GetInt64(config.FlagChainID)

	req := &contract.IdentityRegisterRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse identity registration request"), http.StatusBadRequest)
		return
	}

	registrationStatus, err := te.identityRegistry.GetRegistrationStatus(chainID, id)
	if err != nil {
		log.Err(err).Stack().Msgf("could not check registration status for ID: %s, %+v", id.Address, req)
		utils.SendError(resp, errors.Wrap(err, "could not check registration status"), http.StatusInternalServerError)
		return
	}
	switch registrationStatus {
	case registry.InProgress, registry.Registered:
		log.Info().Msgf("identity %q registration is in status %s, aborting...", id.Address, registrationStatus)
		utils.SendErrorMessage(resp, "Identity already registered", http.StatusConflict)
		return
	}

	regFee := big.NewInt(0)
	if req.ReferralToken == nil {
		rf, err := te.transactor.FetchRegistrationFees(chainID)
		if err != nil {
			utils.SendError(resp, fmt.Errorf("failed to get registration fees %w", err), http.StatusInternalServerError)
			return
		}

		regFee = rf.Fee
	}

	err = te.transactor.RegisterIdentity(id.Address, big.NewInt(0), regFee, "", chainID, req.ReferralToken)
	if err != nil {
		log.Err(err).Msgf("Failed identity registration request for ID: %s, %+v", id.Address, req)
		utils.SendError(resp, errors.Wrap(err, "failed identity registration request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation POST /identities/{id}/beneficiary
// ---
// summary: Settle with Beneficiary
// description: Change beneficiary and settle earnings to it. This is async method.
// parameters:
// - name: id
//   in: path
//   description: Identity address to register
//   type: string
//   required: true
// responses:
//   202:
//     description: settle request accepted
func (te *transactorEndpoint) SettleWithBeneficiaryAsync(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	req := &contract.SettleWithBeneficiaryRequest{}
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, fmt.Errorf("failed to parse set beneficiary request: %w", err), http.StatusBadRequest)
		return
	}

	go func() {
		err = te.bhandler.SettleAndSaveBeneficiary(identity.FromAddress(id), common.HexToAddress(req.Beneficiary))
		if err != nil {
			log.Err(err).Msgf("Failed set beneficiary request for ID: %s, %+v", id, req)
		}
	}()

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation GET /identities/{id}/beneficiary-status
// ---
// summary: Returns beneficiary transaction status
// description: Returns the last beneficiary transaction status for given identity
// parameters:
// - name: id
//   in: path
//   description: Identity address to register
//   type: string
//   required: true
// responses:
//   200:
//     description: Returns beneficiary transaction status
//     schema:
//       "$ref": "#/definitions/BeneficiaryTxStatus"
//   404:
//     description: Beneficiary change never recorded.
func (te *transactorEndpoint) BeneficiaryTxStatus(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	identity := identity.FromAddress(id)
	current, err := te.bprovider.GetBeneficiary(identity.ToCommonAddress())
	if err != nil {
		utils.SendError(resp, fmt.Errorf("failed to parse get current beneficiary: %s", err), http.StatusInternalServerError)
		return
	}
	status, err := te.bhandler.CleanupAndGetChangeStatus(identity, current.Hex())
	if err != nil {
		if errs.Is(err, storm.ErrNotFound) {
			utils.SendError(resp, errors.New("not found"), http.StatusNotFound)
			return
		}
		utils.SendError(resp, fmt.Errorf("failed to get current transaction status: %s", err), http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(
		&contract.BeneficiaryTxStatus{
			State:    status.State,
			Error:    status.Error,
			ChangeTo: status.ChangeTo,
		},
		resp,
	)
}

// swagger:operation GET /settle/history settlementList
// ---
// summary: Returns settlement history
// description: Returns settlement history
// responses:
//   200:
//     description: Returns settlement history
//     schema:
//       "$ref": "#/definitions/SettlementListResponse"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettlementHistory(resp http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	query := contract.NewSettlementListQuery()
	if errors := query.Bind(req); errors.HasErrors() {
		utils.SendValidationErrorMessage(resp, errors)
		return
	}

	settlementsAll, err := te.settlementHistoryProvider.List(query.ToFilter())
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	var settlements []pingpong.SettlementHistoryEntry
	p := utils.NewPaginator(adapter.NewSliceAdapter(settlementsAll), query.PageSize, query.PageSize)
	if err := p.Results(&settlements); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	response := contract.NewSettlementListResponse(settlements, p)
	utils.WriteAsJSON(response, resp)
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
	var req contract.DecreaseStakeRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed to parse decrease stake"), http.StatusBadRequest)
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	fees, err := te.transactor.FetchStakeDecreaseFee(chainID)
	if err != nil {
		utils.SendError(resp, errors.Wrap(err, "failed get stake decrease fee"), http.StatusInternalServerError)
		return
	}

	err = te.transactor.DecreaseStake(req.ID, chainID, req.Amount, fees.Fee)
	if err != nil {
		log.Err(err).Msgf("Failed decreases stake request for ID: %s, %+v", req.ID, req)
		utils.SendError(resp, errors.Wrap(err, "failed decreases stake request"), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

// swagger:operation POST /transactor/settle/withdraw Withdraw
// ---
// summary: Asks to perform withdrawal to l1.
// description: Asks to perform withdrawal to l1.
// parameters:
// - in: body
//   name: body
//   description: withdraw request body
//   schema:
//     $ref: "#/definitions/WithdrawRequestDTO"
// responses:
//   202:
//     description: withdraw request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) Withdraw(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	req := contract.WithdrawRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	err = te.promiseSettler.Withdraw(chainID, identity.FromAddress(req.ProviderID), common.HexToAddress(req.HermesID), common.HexToAddress(req.Beneficiary))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
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
//     $ref: "#/definitions/SettleRequestDTO"
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
//     $ref: "#/definitions/SettleRequestDTO"
// responses:
//   202:
//     description: settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) SettleIntoStakeAsync(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	err := te.settle(request, func(chainID int64, provider identity.Identity, hermes common.Address) error {
		go func() {
			err := te.promiseSettler.SettleIntoStake(chainID, provider, hermes)
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
func AddRoutesForTransactor(
	router *httprouter.Router,
	identityRegistry identityRegistry,
	transactor Transactor,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
	bhandler beneficiarySaver,
	bprovider beneficiaryProvider,
) {
	te := NewTransactorEndpoint(transactor, identityRegistry, promiseSettler, settlementHistoryProvider, addressProvider, bhandler, bprovider)
	router.POST("/identities/:id/register", te.RegisterIdentity)
	router.POST("/identities/:id/beneficiary", te.SettleWithBeneficiaryAsync)
	router.GET("/identities/:id/beneficiary-status", te.BeneficiaryTxStatus)
	router.GET("/transactor/fees", te.TransactorFees)
	router.POST("/transactor/settle/sync", te.SettleSync)
	router.POST("/transactor/settle/async", te.SettleAsync)
	router.GET("/transactor/settle/history", te.SettlementHistory)
	router.POST("/transactor/stake/increase/sync", te.SettleIntoStakeSync)
	router.POST("/transactor/stake/increase/async", te.SettleIntoStakeAsync)
	router.POST("/transactor/stake/decrease", te.DecreaseStake)
	router.POST("/transactor/settle/withdraw", te.Withdraw)
}
