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

	"github.com/spf13/cast"

	"github.com/gin-gonic/gin"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/payout"
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
	RegistrationTokenReward(token string) (*big.Int, error)
}

// promiseSettler settles the given promises
type promiseSettler interface {
	ForceSettle(chainID int64, providerID identity.Identity, hermesID common.Address) error
	GetHermesFee(chainID int64, id common.Address) (uint16, error)
	SettleIntoStake(chainID int64, providerID identity.Identity, hermesID common.Address) error
	Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error
}

type addressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
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
	addressStorage            *payout.AddressStorage
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(
	transactor Transactor,
	identityRegistry identityRegistry,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
) *transactorEndpoint {
	return &transactorEndpoint{
		transactor:                transactor,
		identityRegistry:          identityRegistry,
		promiseSettler:            promiseSettler,
		settlementHistoryProvider: settlementHistoryProvider,
		addressProvider:           addressProvider,
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
func (te *transactorEndpoint) TransactorFees(c *gin.Context) {
	resp := c.Writer

	chainID := config.GetInt64(config.FlagChainID)
	if qcid, err := cast.ToInt64E(c.Query("chain_id")); err == nil {
		chainID = qcid
	}

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
func (te *transactorEndpoint) SettleSync(c *gin.Context) {
	resp := c.Writer
	request := c.Request

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
func (te *transactorEndpoint) SettleAsync(c *gin.Context) {
	resp := c.Writer
	request := c.Request

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
func (te *transactorEndpoint) RegisterIdentity(c *gin.Context) {
	resp := c.Writer
	request := c.Request
	params := c.Params

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
func (te *transactorEndpoint) SettlementHistory(c *gin.Context) {
	resp := c.Writer
	req := c.Request

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
func (te *transactorEndpoint) DecreaseStake(c *gin.Context) {
	resp := c.Writer
	request := c.Request

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
func (te *transactorEndpoint) Withdraw(c *gin.Context) {
	resp := c.Writer
	request := c.Request

	req := contract.WithdrawRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	amount, err := te.parseWithdrawalAmount(req.Amount)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	fromChainID := config.GetInt64(config.FlagChainID)
	if req.FromChainID != 0 {
		if _, ok := registry.Chains()[req.FromChainID]; !ok {
			utils.SendError(resp, errors.New("Unsupported from_chain_id"), http.StatusBadRequest)
			return
		}

		fromChainID = req.FromChainID
	}

	var toChainID int64
	if req.ToChainID != 0 {
		if _, ok := registry.Chains()[req.ToChainID]; !ok {
			utils.SendError(resp, errors.New("Unsupported to_chain_id"), http.StatusBadRequest)
			return
		}

		toChainID = req.ToChainID
	}

	err = te.promiseSettler.Withdraw(fromChainID, toChainID, identity.FromAddress(req.ProviderID), common.HexToAddress(req.HermesID), common.HexToAddress(req.Beneficiary), amount)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

func (te *transactorEndpoint) parseWithdrawalAmount(amount string) (*big.Int, error) {
	if amount == "" {
		return nil, nil
	}

	res, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		return nil, fmt.Errorf("%v is not a valid integer", amount)
	}

	return res, nil
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
func (te *transactorEndpoint) SettleIntoStakeSync(c *gin.Context) {
	resp := c.Writer
	request := c.Request

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
func (te *transactorEndpoint) SettleIntoStakeAsync(c *gin.Context) {
	resp := c.Writer
	request := c.Request

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

// swagger:operation POST /transactor/token/{token}/reward Reward
// ---
// summary: Returns the amount of reward for a token
// parameters:
// - in: path
//   name: token
//   description: Token for which to lookup the reward
//   type: string
//   required: true
// responses:
//   200:
//     description: Token Reward
//     schema:
//       "$ref": "#/definitions/TokenRewardAmount"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (te *transactorEndpoint) TokenRewardAmount(c *gin.Context) {
	resp := c.Writer
	params := c.Params

	token := params.ByName("token")
	reward, err := te.transactor.RegistrationTokenReward(token)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}
	if reward == nil {
		utils.SendError(resp, errors.New("no reward for token"), http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.TokenRewardAmount{
		Amount: reward,
	}, resp)
}

// swagger:operation GET /transactor/chains-summary Chains
// ---
// summary: Returns available chain map
// responses:
//   200:
//     description: Chain Summary
//     schema:
//       "$ref": "#/definitions/ChainSummary"
func (te *transactorEndpoint) ChainSummary(c *gin.Context) {
	chains := registry.Chains()
	result := map[int64]string{}

	for _, id := range []int64{
		config.FlagChain1ChainID.Value,
		config.FlagChain2ChainID.Value,
	} {
		if name, ok := chains[id]; ok {
			result[id] = name
		}
	}

	c.JSON(http.StatusOK, &contract.ChainSummary{
		Chains:       result,
		CurrentChain: config.GetInt64(config.FlagChainID),
	})
}

// AddRoutesForTransactor attaches Transactor endpoints to router
func AddRoutesForTransactor(
	identityRegistry identityRegistry,
	transactor Transactor,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
) func(*gin.Engine) error {
	te := NewTransactorEndpoint(transactor, identityRegistry, promiseSettler, settlementHistoryProvider, addressProvider)

	return func(e *gin.Engine) error {
		idGroup := e.Group("/identities")
		{
			idGroup.POST("/:id/register", te.RegisterIdentity)
		}

		transGroup := e.Group("/transactor")
		{
			transGroup.GET("/fees", te.TransactorFees)
			transGroup.POST("/settle/sync", te.SettleSync)
			transGroup.POST("/settle/async", te.SettleAsync)
			transGroup.GET("/settle/history", te.SettlementHistory)
			transGroup.POST("/stake/increase/sync", te.SettleIntoStakeSync)
			transGroup.POST("/stake/increase/async", te.SettleIntoStakeAsync)
			transGroup.POST("/stake/decrease", te.DecreaseStake)
			transGroup.POST("/settle/withdraw", te.Withdraw)
			transGroup.GET("/token/:token/reward", te.TokenRewardAmount)
			transGroup.GET("/chain-summary", te.ChainSummary)
		}
		return nil
	}
}
