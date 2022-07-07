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

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/shopspring/decimal"

	"github.com/asdine/storm/v3"
	"github.com/spf13/cast"

	"github.com/gin-gonic/gin"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vcraescu/go-paginator/adapter"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/node/core/payout"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// Transactor represents interface to Transactor service
type Transactor interface {
	FetchCombinedFees(chainID int64) (registry.CombinedFeesResponse, error)
	FetchRegistrationFees(chainID int64) (registry.FeesResponse, error)
	FetchSettleFees(chainID int64) (registry.FeesResponse, error)
	FetchStakeDecreaseFee(chainID int64) (registry.FeesResponse, error)
	RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error
	DecreaseStake(id string, chainID int64, amount, transactorFee *big.Int) error
	GetFreeRegistrationEligibility(identity identity.Identity) (bool, error)
	GetFreeProviderRegistrationEligibility() (bool, error)
	OpenChannel(chainID int64, id, hermesID, registryAddress string) error
	ChannelStatus(chainID int64, id, hermesID, registryAddress string) (registry.ChannelStatusResponse, error)
}

// promiseSettler settles the given promises
type promiseSettler interface {
	ForceSettle(chainID int64, providerID identity.Identity, hermesID ...common.Address) error
	SettleIntoStake(chainID int64, providerID identity.Identity, hermesID ...common.Address) error
	GetHermesFee(chainID int64, id common.Address) (uint16, error)
	Withdraw(fromChainID int64, toChainID int64, providerID identity.Identity, hermesID, beneficiary common.Address, amount *big.Int) error
}

type addressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
	GetKnownHermeses(chainID int64) ([]common.Address, error)
	GetHermesChannelAddress(chainID int64, id, hermesAddr common.Address) (common.Address, error)
}

type beneficiarySaver interface {
	SettleAndSaveBeneficiary(id identity.Identity, hermeses []common.Address, beneficiary common.Address) error
	CleanupAndGetChangeStatus(id identity.Identity, currentBeneficiary string) (*beneficiary.ChangeStatus, error)
}

type settlementHistoryProvider interface {
	List(pingpong.SettlementHistoryFilter) ([]pingpong.SettlementHistoryEntry, error)
}

type pilvytisApi interface {
	GetRegistrationPaymentStatus(id identity.Identity) (*pilvytis.RegistrationPaymentResponse, error)
}

type transactorEndpoint struct {
	transactor                Transactor
	affiliator                Affiliator
	identityRegistry          identityRegistry
	promiseSettler            promiseSettler
	settlementHistoryProvider settlementHistoryProvider
	addressProvider           addressProvider
	addressStorage            *payout.AddressStorage
	bprovider                 beneficiaryProvider
	bhandler                  beneficiarySaver
	pilvytis                  pilvytisApi
}

// NewTransactorEndpoint creates and returns transactor endpoint
func NewTransactorEndpoint(
	transactor Transactor,
	identityRegistry identityRegistry,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
	bprovider beneficiaryProvider,
	bhandler beneficiarySaver,
	pilvytis pilvytisApi,
) *transactorEndpoint {
	return &transactorEndpoint{
		transactor:                transactor,
		identityRegistry:          identityRegistry,
		promiseSettler:            promiseSettler,
		settlementHistoryProvider: settlementHistoryProvider,
		addressProvider:           addressProvider,
		bprovider:                 bprovider,
		bhandler:                  bhandler,
		pilvytis:                  pilvytis,
	}
}

// swagger:operation GET /v2/transactor/fees CombinedFeesResponse
// ---
// summary: Returns fees
// description: Returns fees applied by Transactor
// responses:
//   200:
//     description: Fees applied by Transactor
//     schema:
//       "$ref": "#/definitions/CombinedFeesResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) TransactorFeesV2(c *gin.Context) {
	chainID := config.GetInt64(config.FlagChainID)
	if qcid, err := cast.ToInt64E(c.Query("chain_id")); err == nil {
		chainID = qcid
	}

	fees, err := te.transactor.FetchCombinedFees(chainID)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to fetch fees", contract.ErrCodeTransactorFetchFees))
		return
	}

	hermes, err := te.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	hermesFeePerMyriad, err := te.promiseSettler.GetHermesFee(chainID, hermes)
	if err != nil {
		c.Error(apierror.Internal("Could not get hermes fee: "+err.Error(), contract.ErrCodeHermesFee))
		return
	}

	hermesPercent := decimal.NewFromInt(int64(hermesFeePerMyriad)).Div(decimal.NewFromInt(10000))
	f := contract.CombinedFeesResponse{
		Current:    contract.NewTransactorFees(&fees.Current),
		Last:       contract.NewTransactorFees(&fees.Last),
		ServerTime: fees.ServerTime,

		HermesPercent: hermesPercent.StringFixed(4),
	}

	utils.WriteAsJSON(f, c.Writer)
}

// swagger:operation GET /transactor/fees FeesDTO
// ---
// summary: Returns fees
// deprecated: true
// description: Returns fees applied by Transactor
// responses:
//   200:
//     description: Fees applied by Transactor
//     schema:
//       "$ref": "#/definitions/FeesDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) TransactorFees(c *gin.Context) {
	chainID := config.GetInt64(config.FlagChainID)
	if qcid, err := cast.ToInt64E(c.Query("chain_id")); err == nil {
		chainID = qcid
	}

	registrationFees, err := te.transactor.FetchRegistrationFees(chainID)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to fetch fees", contract.ErrCodeTransactorFetchFees))
		return
	}
	settlementFees, err := te.transactor.FetchSettleFees(chainID)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to fetch fees", contract.ErrCodeTransactorFetchFees))
		return
	}
	decreaseStakeFees, err := te.transactor.FetchStakeDecreaseFee(chainID)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to fetch fees", contract.ErrCodeTransactorFetchFees))
		return
	}

	hermes, err := te.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	hermesFeePerMyriad, err := te.promiseSettler.GetHermesFee(chainID, hermes)
	if err != nil {
		c.Error(apierror.Internal("Could not get hermes fee: "+err.Error(), contract.ErrCodeHermesFee))
		return
	}

	hermesPercent := decimal.NewFromInt(int64(hermesFeePerMyriad)).Div(decimal.NewFromInt(10000))
	f := contract.FeesDTO{
		Registration:        registrationFees.Fee,
		RegistrationTokens:  contract.NewTokens(registrationFees.Fee),
		Settlement:          settlementFees.Fee,
		SettlementTokens:    contract.NewTokens(settlementFees.Fee),
		HermesPercent:       hermesPercent.StringFixed(4),
		Hermes:              hermesFeePerMyriad,
		DecreaseStake:       decreaseStakeFees.Fee,
		DecreaseStakeTokens: contract.NewTokens(decreaseStakeFees.Fee),
	}

	utils.WriteAsJSON(f, c.Writer)
}

// swagger:operation POST /transactor/settle/sync SettleSync
// ---
// summary: Forces the settlement of promises for the given provider and hermes
// description: Forces a settlement for the hermes promises and blocks until the settlement is complete.
// parameters:
// - in: body
//   name: body
//   description: Settle request
//   schema:
//     $ref: "#/definitions/SettleRequestDTO"
// responses:
//   202:
//     description: Settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettleSync(c *gin.Context) {
	err := te.settle(c.Request, te.promiseSettler.ForceSettle)
	if err != nil {
		log.Err(err).Msg("Settle failed")
		utils.ForwardError(c, err, apierror.Internal("Could not force settle", contract.ErrCodeHermesSettle))
		return
	}
	c.Status(http.StatusOK)
}

// swagger:operation POST /transactor/settle/async SettleAsync
// ---
// summary: forces the settlement of promises for the given provider and hermes
// description: Forces a settlement for the hermes promises. Does not wait for completion.
// parameters:
// - in: body
//   name: body
//   description: Settle request
//   schema:
//     $ref: "#/definitions/SettleRequestDTO"
// responses:
//   202:
//     description: Settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettleAsync(c *gin.Context) {
	err := te.settle(c.Request, func(chainID int64, provider identity.Identity, hermes ...common.Address) error {
		providerId := provider.ToCommonAddress()
		beneficiary, err := te.bprovider.GetBeneficiary(providerId)
		if err != nil {
			return fmt.Errorf("failed to get beneficiary: %w", err)
		}
		isChannel, err := isBenenficiarySetToChannel(te.addressProvider, chainID, providerId, beneficiary)
		if err != nil {
			return fmt.Errorf("failed to check if channel is set as beneficiary: %w", err)
		}
		if isChannel {
			return fmt.Errorf("payment channel is set as beneficiary, please turn on auto-withdrawals using your personal wallet address")
		}
		go func() {
			err := te.promiseSettler.ForceSettle(chainID, provider, hermes...)
			if err != nil {
				log.Error().Err(err).Msgf("Could not settle provider(%q) promises", provider.Address)
			}
		}()
		return nil
	})
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to force settle async", contract.ErrCodeHermesSettleAsync))
		return
	}

	c.Status(http.StatusAccepted)
}

func (te *transactorEndpoint) settle(request *http.Request, settler func(int64, identity.Identity, ...common.Address) error) error {
	req := contract.SettleRequest{}

	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal settle request")
	}

	hermesIDs := []common.Address{
		// TODO: Remove this ant just use req.HermesIDs when UI upgrades.
		common.HexToAddress(req.HermesID),
	}

	if len(req.HermesIDs) > 0 {
		hermesIDs = req.HermesIDs
	}

	if len(hermesIDs) == 0 {
		return errors.New("must specify a hermes to settle with")
	}

	chainID := config.GetInt64(config.FlagChainID)
	return settler(chainID, identity.FromAddress(req.ProviderID), hermesIDs...)
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
//     description: Identity registered.
//   202:
//     description: Identity registration accepted and will be processed.
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   422:
//     description: Unable to process the request at this point
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) RegisterIdentity(c *gin.Context) {
	id := identity.FromAddress(c.Param("id"))
	chainID := config.GetInt64(config.FlagChainID)

	req := &contract.IdentityRegisterRequest{}

	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	registrationStatus, err := te.identityRegistry.GetRegistrationStatus(chainID, id)
	if err != nil {
		log.Err(err).Stack().Msgf("Could not check registration status for ID: %s, %+v", id.Address, req)
		c.Error(apierror.Internal(fmt.Sprintf("could not check registration status for ID: %s", id.Address), contract.ErrCodeIDBlockchainRegistrationCheck))
		return
	}
	switch registrationStatus {
	case registry.InProgress:
		log.Info().Msgf("Identity %q registration is in status %s, aborting...", id.Address, registrationStatus)
		c.Error(apierror.Unprocessable("Identity registration in progress", contract.ErrCodeIDRegistrationInProgress))
		return
	case registry.Registered:
		c.Status(http.StatusOK)
		return
	}

	regFee := big.NewInt(0)
	if !te.canRegisterForFree(req, id) {
		if req.Fee == nil || req.Fee.Cmp(big.NewInt(0)) == 0 {
			rf, err := te.transactor.FetchRegistrationFees(chainID)
			if err != nil {
				utils.ForwardError(c, err, apierror.Internal("Failed to fetch fees", contract.ErrCodeTransactorFetchFees))
				return
			}

			regFee = rf.Fee
		} else {
			regFee = req.Fee
		}
	}

	err = te.transactor.RegisterIdentity(id.Address, big.NewInt(0), regFee, req.Beneficiary, chainID, req.ReferralToken)
	if err != nil {
		log.Err(err).Msgf("Failed identity registration request for ID: %s, %+v", id.Address, req)
		utils.ForwardError(c, err, apierror.Internal("Failed to register identity", contract.ErrCodeTransactorRegistration))
		return
	}

	c.Status(http.StatusAccepted)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettlementHistory(c *gin.Context) {
	query := contract.NewSettlementListQuery()
	if err := query.Bind(c.Request); err != nil {
		c.Error(err)
		return
	}

	settlementsAll, err := te.settlementHistoryProvider.List(query.ToFilter())
	if err != nil {
		c.Error(apierror.Internal("Could not list settlement history: "+err.Error(), contract.ErrCodeTransactorSettleHistory))
		return
	}

	var pagedSettlements []pingpong.SettlementHistoryEntry
	p := utils.NewPaginator(adapter.NewSliceAdapter(settlementsAll), query.PageSize, query.Page)
	if err := p.Results(&pagedSettlements); err != nil {
		c.Error(apierror.Internal("Could not paginate settlement history: "+err.Error(), contract.ErrCodeTransactorSettleHistoryPaginate))
		return
	}

	withdrawalTotal := big.NewInt(0)
	for _, s := range settlementsAll {
		if s.IsWithdrawal {
			withdrawalTotal.Add(withdrawalTotal, s.Amount)
		}
	}

	response := contract.NewSettlementListResponse(withdrawalTotal, pagedSettlements, p)
	utils.WriteAsJSON(response, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) DecreaseStake(c *gin.Context) {
	var req contract.DecreaseStakeRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	fees, err := te.transactor.FetchStakeDecreaseFee(chainID)
	if err != nil {
		c.Error(err)
		return
	}

	err = te.transactor.DecreaseStake(req.ID, chainID, req.Amount, fees.Fee)
	if err != nil {
		log.Err(err).Msgf("Failed decreases stake request for ID: %s, %+v", req.ID, req)
		utils.ForwardError(c, err, apierror.Internal("Failed to decrease stake: "+err.Error(), contract.ErrCodeTransactorDecreaseStake))
		return
	}

	c.Status(http.StatusAccepted)
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
//     description: Withdraw request accepted
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) Withdraw(c *gin.Context) {
	var req contract.WithdrawRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	amount, err := req.AmountInMYST()
	if err != nil {
		c.Error(apierror.BadRequestField("'amount' is invalid", apierror.ValidateErrInvalidVal, "amount"))
		return
	}

	fromChainID := config.GetInt64(config.FlagChainID)
	if req.FromChainID != 0 {
		if _, ok := registry.Chains()[req.FromChainID]; !ok {
			c.Error(apierror.BadRequestField("Unsupported from_chain_id", apierror.ValidateErrInvalidVal, "from_chain_id"))
			return
		}

		fromChainID = req.FromChainID
	}

	var toChainID int64
	if req.ToChainID != 0 {
		if _, ok := registry.Chains()[req.ToChainID]; !ok {
			c.Error(apierror.BadRequestField("Unsupported to_chain_id", apierror.ValidateErrInvalidVal, "to_chain_id"))
			return
		}

		toChainID = req.ToChainID
	}

	err = te.promiseSettler.Withdraw(fromChainID, toChainID, identity.FromAddress(req.ProviderID), common.HexToAddress(req.HermesID), common.HexToAddress(req.Beneficiary), amount)
	if err != nil {
		log.Err(err).Fields(map[string]interface{}{
			"from_chain_id": fromChainID,
			"to_chain_id":   toChainID,
			"provider_id":   req.ProviderID,
			"hermes_id":     req.HermesID,
			"beneficiary":   req.Beneficiary,
			"amount":        amount.String(),
		}).Msg("Withdrawal failed")
		utils.ForwardError(c, err, apierror.Internal("Could not withdraw", contract.ErrCodeTransactorWithdraw))
		return
	}

	c.Status(http.StatusOK)
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
// summary: Forces the settlement with stake increase of promises for the given provider and hermes.
// description: Forces a settlement with stake increase for the hermes promises and blocks until the settlement is complete.
// parameters:
// - in: body
//   name: body
//   description: Settle request
//   schema:
//     $ref: "#/definitions/SettleRequestDTO"
// responses:
//   202:
//     description: Settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettleIntoStakeSync(c *gin.Context) {
	err := te.settle(c.Request, te.promiseSettler.SettleIntoStake)
	if err != nil {
		log.Err(err).Msg("Settle into stake failed")
		utils.ForwardError(c, err, apierror.Internal("Could not settle into stake", contract.ErrCodeTransactorSettle))
		return
	}

	c.Status(http.StatusOK)
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
//     description: Settle request accepted
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettleIntoStakeAsync(c *gin.Context) {
	err := te.settle(c.Request, func(chainID int64, provider identity.Identity, hermes ...common.Address) error {
		go func() {
			err := te.promiseSettler.SettleIntoStake(chainID, provider, hermes...)
			if err != nil {
				log.Error().Err(err).Msgf("could not settle into stake provider(%q) promises", provider.Address)
			}
		}()
		return nil
	})
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Could not settle into stake async", contract.ErrCodeTransactorSettle))
		return
	}

	c.Status(http.StatusOK)
}

// swagger:operation POST /transactor/token/{token}/reward Reward
// ---
// summary: Returns the amount of reward for a token
// deprecated: true
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
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) TokenRewardAmount(c *gin.Context) {
	token := c.Param("token")
	reward, err := te.affiliator.RegistrationTokenReward(token)
	if err != nil {
		c.Error(err)
		return
	}
	if reward == nil {
		c.Error(apierror.Internal("No reward for token", contract.ErrCodeTransactorNoReward))
		return
	}

	utils.WriteAsJSON(contract.TokenRewardAmount{
		Amount: reward,
	}, c.Writer)
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

// EligibilityResponse represents the eligibility response
// swagger:model EligibilityResponse
type EligibilityResponse struct {
	Eligible bool `json:"eligible"`
}

// swagger:operation GET /transactor/identities/{id}/eligibility Eligibility
// ---
// summary: Checks if given id is eligible for free registration
// parameters:
// - name: id
//   in: path
//   description: Identity address to register
//   type: string
//   required: true
// responses:
//   200:
//     description: Eligibility response
//     schema:
//       "$ref": "#/definitions/EligibilityResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) FreeRegistrationEligibility(c *gin.Context) {
	id := identity.FromAddress(c.Param("id"))

	res, err := te.transactor.GetFreeRegistrationEligibility(id)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to check if free registration is possible", contract.ErrCodeTransactorRegistration))
		return
	}

	c.JSON(http.StatusOK, EligibilityResponse{Eligible: res})
}

// swagger:operation GET /identities/provider/eligibility ProviderEligibility
// ---
// summary: Checks if provider is eligible for free registration
// responses:
//   200:
//     description: Eligibility response
//     schema:
//       "$ref": "#/definitions/EligibilityResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) FreeProviderRegistrationEligibility(c *gin.Context) {
	res, err := te.transactor.GetFreeProviderRegistrationEligibility()
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to check if free registration is possible", contract.ErrCodeTransactorRegistration))
		return
	}

	c.JSON(http.StatusOK, EligibilityResponse{Eligible: res})
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
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) BeneficiaryTxStatus(c *gin.Context) {
	id := identity.FromAddress(c.Param("id"))
	current, err := te.bprovider.GetBeneficiary(id.ToCommonAddress())
	if err != nil {
		c.Error(apierror.Internal("Failed to get current beneficiary: "+err.Error(), contract.ErrCodeTransactorBeneficiary))
		return
	}
	status, err := te.bhandler.CleanupAndGetChangeStatus(id, current.Hex())
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Error(apierror.Internal("Failed to get current transaction status: "+err.Error(), contract.ErrCodeTransactorBeneficiaryTxStatus))
		return
	}

	utils.WriteAsJSON(
		&contract.BeneficiaryTxStatus{
			State:    status.State,
			Error:    status.Error,
			ChangeTo: status.ChangeTo,
		},
		c.Writer,
	)
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
//     description: Settle request accepted
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
func (te *transactorEndpoint) SettleWithBeneficiaryAsync(c *gin.Context) {
	id := c.Param("id")

	req := &contract.SettleWithBeneficiaryRequest{}
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	chainID := config.GetInt64(config.FlagChainID)

	hermesID := common.HexToAddress(req.HermesID)
	hermeses := []common.Address{hermesID}
	if hermesID == common.HexToAddress("") {
		hermeses, err = te.addressProvider.GetKnownHermeses(chainID)
		if err != nil {
			c.Error(err)
			return
		}
	}

	go func() {
		err = te.bhandler.SettleAndSaveBeneficiary(identity.FromAddress(id), hermeses, common.HexToAddress(req.Beneficiary))
		if err != nil {
			log.Err(err).Msgf("Failed set beneficiary request for ID: %s, %+v", id, req)
		}
	}()

	c.Status(http.StatusAccepted)
}

// AddRoutesForTransactor attaches Transactor endpoints to router
func AddRoutesForTransactor(
	identityRegistry identityRegistry,
	transactor Transactor,
	affiliator Affiliator,
	promiseSettler promiseSettler,
	settlementHistoryProvider settlementHistoryProvider,
	addressProvider addressProvider,
	bprovider beneficiaryProvider,
	bhandler beneficiarySaver,
	pilvytis pilvytisApi,
) func(*gin.Engine) error {
	te := NewTransactorEndpoint(transactor, identityRegistry, promiseSettler, settlementHistoryProvider, addressProvider, bprovider, bhandler, pilvytis)
	a := NewAffiliatorEndpoint(affiliator)

	return func(e *gin.Engine) error {
		idGroup := e.Group("/identities")
		{
			idGroup.POST("/:id/register", te.RegisterIdentity)
			idGroup.GET("/provider/eligibility", te.FreeProviderRegistrationEligibility)
			idGroup.GET("/:id/eligibility", te.FreeRegistrationEligibility)
			idGroup.GET("/:id/beneficiary-status", te.BeneficiaryTxStatus)
			idGroup.POST("/:id/beneficiary", te.SettleWithBeneficiaryAsync)
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
			transGroup.GET("/token/:token/reward", a.TokenRewardAmount)
			transGroup.GET("/chain-summary", te.ChainSummary)
		}
		transGroupV2 := e.Group("/v2/transactor")
		{
			transGroupV2.GET("/fees", te.TransactorFeesV2)
		}
		return nil
	}
}

func (te *transactorEndpoint) canRegisterForFree(req *contract.IdentityRegisterRequest, id identity.Identity) bool {
	if req.ReferralToken != nil {
		return true
	}
	resp, err := te.pilvytis.GetRegistrationPaymentStatus(id)
	if err != nil {
		log.Warn().AnErr("err", err).Msg("Failed to get registration payment status from pilvytis")
		return false
	}
	return resp.Paid
}
