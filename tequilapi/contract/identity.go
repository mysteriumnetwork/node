/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/identity"
	pingpong_event "github.com/mysteriumnetwork/node/session/pingpong/event"
)

// IdentityRefDTO represents unique identity reference.
// swagger:model IdentityRefDTO
type IdentityRefDTO struct {
	// identity in Ethereum address format
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	Address string `json:"id"`
}

// BalanceDTO holds balance information.
// swagger:model BalanceDTO
type BalanceDTO struct {
	Balance       *big.Int `json:"balance"`
	BalanceTokens Tokens   `json:"balance_tokens"`
}

// IdentityDTO holds identity information.
// swagger:model IdentityDTO
type IdentityDTO struct {
	// identity in Ethereum address format
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	Address            string `json:"id"`
	RegistrationStatus string `json:"registration_status"`
	ChannelAddress     string `json:"channel_address"`

	// deprecated
	Balance       *big.Int `json:"balance"`
	Earnings      *big.Int `json:"earnings"`
	EarningsTotal *big.Int `json:"earnings_total"`
	// ===========

	BalanceTokens       Tokens `json:"balance_tokens"`
	EarningsTokens      Tokens `json:"earnings_tokens"`
	EarningsTotalTokens Tokens `json:"earnings_total_tokens"`

	Stake             *big.Int               `json:"stake"`
	HermesID          string                 `json:"hermes_id"`
	EarningsPerHermes map[string]EarningsDTO `json:"earnings_per_hermes"`
}

// EarningsDTO holds earnings data.
// swagger:model EarningsDTO
type EarningsDTO struct {
	Earnings      Tokens `json:"earnings"`
	EarningsTotal Tokens `json:"earnings_total"`
}

// NewEarningsPerHermesDTO transforms the pingong value in a public one.
func NewEarningsPerHermesDTO(earnings map[common.Address]pingpong_event.Earnings) map[string]EarningsDTO {
	settlementsPerHermes := make(map[string]EarningsDTO)
	for h, earn := range earnings {
		settlementsPerHermes[h.Hex()] = EarningsDTO{
			Earnings:      NewTokens(earn.UnsettledBalance),
			EarningsTotal: NewTokens(earn.LifetimeBalance),
		}
	}

	return settlementsPerHermes
}

// NewIdentityDTO maps to API identity.
func NewIdentityDTO(id identity.Identity) IdentityRefDTO {
	return IdentityRefDTO{Address: id.Address}
}

// ListIdentitiesResponse holds list of identities.
// swagger:model ListIdentitiesResponse
type ListIdentitiesResponse struct {
	Identities []IdentityRefDTO `json:"identities"`
}

// NewIdentityListResponse maps to API identity list.
func NewIdentityListResponse(ids []identity.Identity) ListIdentitiesResponse {
	result := ListIdentitiesResponse{
		Identities: make([]IdentityRefDTO, len(ids)),
	}
	for i, id := range ids {
		result.Identities[i] = NewIdentityDTO(id)
	}
	return result
}

// IdentityCreateRequest request used for new identity creation.
// swagger:model IdentityCreateRequestDTO
type IdentityCreateRequest struct {
	Passphrase *string `json:"passphrase"`
}

// Validate validates fields in request
func (r IdentityCreateRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if r.Passphrase == nil {
		v.Required("passphrase")
	}
	return v.Err()
}

// IdentityUnlockRequest request used for identity unlocking.
// swagger:model IdentityUnlockRequestDTO
type IdentityUnlockRequest struct {
	Passphrase *string `json:"passphrase"`
}

// Validate validates fields in request
func (r IdentityUnlockRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if r.Passphrase == nil {
		v.Required("passphrase")
	}
	return v.Err()
}

// IdentityCurrentRequest request used for current identity remembering.
// swagger:model IdentityCurrentRequestDTO
type IdentityCurrentRequest struct {
	Address    *string `json:"id"`
	Passphrase *string `json:"passphrase"`
}

// Validate validates fields in request
func (r IdentityCurrentRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if r.Passphrase == nil {
		v.Required("passphrase")
	}
	return v.Err()
}

// IdentityRegisterRequest represents the identity registration user input parameters
// swagger:model IdentityRegisterRequestDTO
type IdentityRegisterRequest struct {
	// Token: referral token, if the user has one
	ReferralToken *string `json:"referral_token,omitempty"`
	// Beneficiary: beneficiary to set during registration. Optional.
	Beneficiary string `json:"beneficiary,omitempty"`
	// Fee: an agreed amount to pay for registration
	Fee *big.Int `json:"fee"`
}

// IdentityRegistrationResponse represents registration status and needed data for registering of given identity
// swagger:model IdentityRegistrationResponseDTO
type IdentityRegistrationResponse struct {
	Status string `json:"status"`
	// Returns true if identity is registered in payments smart contract
	Registered bool `json:"registered"`
}

// IdentityBeneficiaryResponse represents the provider beneficiary address.
// swagger:model IdentityBeneficiaryResponseDTO
type IdentityBeneficiaryResponse struct {
	Beneficiary      string `json:"beneficiary"`
	IsChannelAddress bool   `json:"is_channel_address"`
}

// IdentityImportRequest is received in identity import endpoint.
// swagger:model IdentityImportRequest
type IdentityImportRequest struct {
	Data              []byte `json:"data"`
	CurrentPassphrase string `json:"current_passphrase,omitempty"`

	// Optional. Default values are OK.
	SetDefault    bool   `json:"set_default"`
	NewPassphrase string `json:"new_passphrase"`
}

// Validate validates the import request.
func (i *IdentityImportRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if len(i.CurrentPassphrase) == 0 {
		v.Required("current_passphrase")
	}
	if len(i.Data) == 0 {
		v.Required("data")
	}
	return v.Err()
}

// IdentityExportRequest is received in identity export endpoint.
// swagger:model IdentityExportRequestDTO
type IdentityExportRequest struct {
	Identity      string `json:"identity,omitempty"`
	NewPassphrase string `json:"newpassphrase,omitempty"`
}

// Validate validates the Export request.
func (i *IdentityExportRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	if len(i.NewPassphrase) == 0 {
		v.Required("newpassphrase")
	}
	return v.Err()
}

// borrowed from github.com/ethereum/go-ethereum@v1.10.17/accounts/keystore/key.go

// EncryptedKeyJSON represents response to IdentityExportRequest.
// swagger:model IdentityExportResponseDTO
type EncryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// BeneficiaryAddressRequest address of the beneficiary
// swagger:model BeneficiaryAddressRequest
type BeneficiaryAddressRequest struct {
	Address string `json:"address"`
}
