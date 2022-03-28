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
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/payout"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	pingpong_event "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type balanceProvider interface {
	GetBalance(chainID int64, id identity.Identity) *big.Int
	ForceBalanceUpdateCached(chainID int64, id identity.Identity) *big.Int
}

type earningsProvider interface {
	GetEarningsDetailed(chainID int64, id identity.Identity) *pingpong_event.EarningsDetailed
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
	channelCalculator AddressProvider
	balanceProvider   balanceProvider
	earningsProvider  earningsProvider
	bc                providerChannel
	transactor        Transactor
	bprovider         beneficiaryProvider
	addressStorage    *payout.AddressStorage
}

// AddressProvider provides sc addresses.
type AddressProvider interface {
	GetActiveHermes(chainID int64) (common.Address, error)
	GetChannelAddress(chainID int64, id common.Address) (common.Address, error)
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
func (ia *identitiesAPI) List(c *gin.Context) {
	ids := ia.idm.GetIdentities()
	idsDTO := contract.NewIdentityListResponse(ids)
	utils.WriteAsJSON(idsDTO, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) Current(c *gin.Context) {
	var req contract.IdentityCurrentRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	idAddress := ""
	if req.Address != nil {
		idAddress = *req.Address
	}

	chainID := config.GetInt64(config.FlagChainID)
	id, err := ia.selector.UseOrCreate(idAddress, *req.Passphrase, chainID)

	if err != nil {
		c.Error(apierror.Internal("Failed to use/create ID: "+err.Error(), contract.ErrCodeIDUseOrCreate))
		return
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) Create(c *gin.Context) {
	var req contract.IdentityCreateRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	id, err := ia.idm.CreateNewIdentity(*req.Passphrase)
	if err != nil {
		c.Error(apierror.Internal("Failed to create ID", contract.ErrCodeIDCreate))
		return
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   403:
//     description: Unlock failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   404:
//     description: ID not found
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) Unlock(c *gin.Context) {
	address := c.Param("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		c.Error(apierror.NotFound("ID not found"))
		return
	}

	var req contract.IdentityUnlockRequest
	err = json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	err = ia.idm.Unlock(chainID, id.Address, *req.Passphrase)
	if err != nil {
		c.Error(apierror.Forbidden("Unlock failed", contract.ErrCodeIDUnlock))
		return
	}
	c.Status(http.StatusAccepted)
}

// swagger:operation PUT /identities/{id}/balance/refresh Identity balance
// ---
// summary: Refresh balance of given identity
// description: Refresh balance of given identity
// parameters:
//   - in: path
//     name: id
//     description: hex address of identity
//     type: string
//     required: true
// responses:
//   200:
//     description: Updated balance
//     schema:
//       "$ref": "#/definitions/BalanceDTO"
//   404:
//     description: ID not found
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) BalanceRefresh(c *gin.Context) {
	address := c.Param("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		c.Error(apierror.NotFound("Identity not found"))
		return
	}
	chainID := config.GetInt64(config.FlagChainID)
	balance := ia.balanceProvider.ForceBalanceUpdateCached(chainID, id)
	status := contract.BalanceDTO{
		Balance:       balance,
		BalanceTokens: contract.NewTokens(balance),
	}
	utils.WriteAsJSON(status, c.Writer)
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
//   404:
//     description: ID not found
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) Get(c *gin.Context) {
	address := c.Param("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		c.Error(apierror.NotFound("Identity not found"))
		return
	}

	chainID := config.GetInt64(config.FlagChainID)
	regStatus, err := ia.registry.GetRegistrationStatus(chainID, id)
	if err != nil {
		c.Error(apierror.Internal("Failed to check ID registration status: "+err.Error(), contract.ErrCodeIDRegistrationCheck))
		return
	}

	channelAddress, err := ia.channelCalculator.GetChannelAddress(chainID, id.ToCommonAddress())
	if err != nil {
		c.Error(apierror.Internal("Failed to calculate channel address: "+err.Error(), contract.ErrCodeIDCalculateAddress))
		return
	}

	defaultHermesID, err := ia.channelCalculator.GetActiveHermes(chainID)
	if err != nil {
		c.Error(apierror.Internal("Could not get active hermes: "+err.Error(), contract.ErrCodeActiveHermes))
		return
	}

	var stake = new(big.Int)
	if regStatus == registry.Registered {
		data, err := ia.bc.GetProviderChannel(chainID, defaultHermesID, common.HexToAddress(address), false)
		if err != nil {
			c.Error(apierror.Internal("Failed to check blockchain registration status: "+err.Error(), contract.ErrCodeIDBlockchainRegistrationCheck))
			return
		}
		stake = data.Stake
	}

	balance := ia.balanceProvider.GetBalance(chainID, id)
	earnings := ia.earningsProvider.GetEarningsDetailed(chainID, id)

	settlementsPerHermes := make(map[string]contract.EarningsDTO)
	for h, earn := range earnings.PerHermes {
		settlementsPerHermes[h.Hex()] = contract.EarningsDTO{
			Earnings:      contract.NewTokens(earn.UnsettledBalance),
			EarningsTotal: contract.NewTokens(earn.LifetimeBalance),
		}
	}

	status := contract.IdentityDTO{
		Address:             address,
		RegistrationStatus:  regStatus.String(),
		ChannelAddress:      channelAddress.Hex(),
		Balance:             balance,
		BalanceTokens:       contract.NewTokens(balance),
		Earnings:            earnings.Total.UnsettledBalance,
		EarningsTokens:      contract.NewTokens(earnings.Total.UnsettledBalance),
		EarningsTotal:       earnings.Total.LifetimeBalance,
		EarningsTotalTokens: contract.NewTokens(earnings.Total.LifetimeBalance),
		Stake:               stake,
		HermesID:            defaultHermesID.Hex(),
		EarningsPerHermes:   contract.NewEarningsPerHermesDTO(earnings.PerHermes),
	}
	utils.WriteAsJSON(status, c.Writer)
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
//   404:
//     description: ID not found
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) RegistrationStatus(c *gin.Context) {
	address := c.Param("id")
	id, err := ia.idm.GetIdentity(address)
	if err != nil {
		c.Error(apierror.NotFound("ID not found"))
		return
	}

	regStatus, err := ia.registry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), id)
	if err != nil {
		c.Error(apierror.Internal("Failed to check ID registration status", contract.ErrCodeIDRegistrationCheck))
		return
	}

	registrationDataDTO := &contract.IdentityRegistrationResponse{
		Status:     regStatus.String(),
		Registered: regStatus.Registered(),
	}
	utils.WriteAsJSON(registrationDataDTO, c.Writer)
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
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) Beneficiary(c *gin.Context) {
	address := c.Param("id")
	data, err := ia.bprovider.GetBeneficiary(common.HexToAddress(address))
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get beneficiary address", contract.ErrCodeBeneficiaryGet))
		return
	}

	registrationDataDTO := &contract.IdentityBeneficiaryResponse{
		Beneficiary: data.Hex(),
	}
	utils.WriteAsJSON(registrationDataDTO, c.Writer)
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
func (ia *identitiesAPI) Import(c *gin.Context) {
	var req contract.IdentityImportRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := req.Validate(); err != nil {
		c.Error(err)
		return
	}

	id, err := ia.mover.Import(req.Data, req.CurrentPassphrase, req.NewPassphrase)
	if err != nil {
		c.Error(apierror.Unprocessable(fmt.Sprintf("Failed to import identity: %s", err), contract.ErrCodeIDImport))
		return
	}

	if req.SetDefault {
		if err := ia.selector.SetDefault(id.Address); err != nil {
			c.Error(apierror.Unprocessable(fmt.Sprintf("Failed to set default identity: %s", err), contract.ErrCodeIDSetDefault))
			return
		}
	}

	idDTO := contract.NewIdentityDTO(id)
	utils.WriteAsJSON(idDTO, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) GetPayoutAddress(c *gin.Context) {
	id := c.Param("id")
	addr, err := ia.addressStorage.Address(id)
	if err != nil {
		if errors.Is(err, payout.ErrNotFound) {
			utils.WriteAsJSON(contract.PayoutAddressRequest{}, c.Writer)
			return
		}
		c.Error(apierror.Internal("Failed to get payout address", contract.ErrCodeIDGetPayoutAddress))
		return
	}

	utils.WriteAsJSON(contract.PayoutAddressRequest{Address: addr}, c.Writer)
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
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
func (ia *identitiesAPI) SavePayoutAddress(c *gin.Context) {
	var par contract.PayoutAddressRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&par); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	id := c.Param("id")
	err := ia.addressStorage.Save(id, par.Address)
	if err != nil {
		c.Error(apierror.BadRequest("Invalid address", contract.ErrCodeIDSavePayoutAddress))
		return
	}

	utils.WriteAsJSON(par, c.Writer)
}

// AddRoutesForIdentities creates /identities endpoint on tequilapi service
func AddRoutesForIdentities(
	idm identity.Manager,
	selector identity_selector.Handler,
	registry registry.IdentityRegistry,
	balanceProvider balanceProvider,
	channelAddressCalculator *client.MultiChainAddressProvider,
	earningsProvider earningsProvider,
	bc providerChannel,
	transactor Transactor,
	bprovider beneficiaryProvider,
	mover identityMover,
	addressStorage *payout.AddressStorage,
) func(*gin.Engine) error {
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
	return func(e *gin.Engine) error {
		identityGroup := e.Group("/identities")
		{
			identityGroup.GET("", idAPI.List)
			identityGroup.POST("", idAPI.Create)
			identityGroup.PUT("/current", idAPI.Current)
			identityGroup.GET("/:id", idAPI.Get)
			identityGroup.GET("/:id/status", idAPI.Get)
			identityGroup.PUT("/:id/unlock", idAPI.Unlock)
			identityGroup.GET("/:id/registration", idAPI.RegistrationStatus)
			identityGroup.GET("/:id/beneficiary", idAPI.Beneficiary)
			identityGroup.GET("/:id/payout-address", idAPI.GetPayoutAddress)
			identityGroup.PUT("/:id/payout-address", idAPI.SavePayoutAddress)
			identityGroup.PUT("/:id/balance/refresh", idAPI.BalanceRefresh)
		}
		e.POST("/identities-import", idAPI.Import)
		return nil
	}
}
