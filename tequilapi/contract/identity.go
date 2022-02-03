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
	"errors"
	"math/big"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
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
	Balance *big.Int
}

// IdentityDTO holds identity information.
// swagger:model IdentityDTO
type IdentityDTO struct {
	// identity in Ethereum address format
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	Address            string   `json:"id"`
	RegistrationStatus string   `json:"registration_status"`
	ChannelAddress     string   `json:"channel_address"`
	Balance            *big.Int `json:"balance"`
	BalanceTokens      Tokens   `json:"balance_tokens"`
	Earnings           *big.Int `json:"earnings"`
	EarningsTotal      *big.Int `json:"earnings_total"`
	Stake              *big.Int `json:"stake"`
	HermesID           string   `json:"hermes_id"`
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
func (r IdentityCreateRequest) Validate() *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if r.Passphrase == nil {
		errors.ForField("passphrase").Required()
	}
	return errors
}

// IdentityUnlockRequest request used for identity unlocking.
// swagger:model IdentityUnlockRequestDTO
type IdentityUnlockRequest struct {
	Passphrase *string `json:"passphrase"`
}

// Validate validates fields in request
func (r IdentityUnlockRequest) Validate() *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if r.Passphrase == nil {
		errors.ForField("passphrase").Required()
	}
	return errors
}

// IdentityCurrentRequest request used for current identity remembering.
// swagger:model IdentityCurrentRequestDTO
type IdentityCurrentRequest struct {
	Address    *string `json:"id"`
	Passphrase *string `json:"passphrase"`
}

// Validate validates fields in request
func (r IdentityCurrentRequest) Validate() *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if r.Passphrase == nil {
		errors.ForField("passphrase").Required()
	}
	return errors
}

// IdentityRegisterRequest represents the identity registration user input parameters
// swagger:model IdentityRegisterRequestDTO
type IdentityRegisterRequest struct {
	// Token: referral token, if the user has one
	ReferralToken *string `json:"referral_token,omitempty"`
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
	Beneficiary string `json:"beneficiary"`
}

// IdentityImportRequest is received in identity import endpoint.
//swagger:model IdentityImportRequest
type IdentityImportRequest struct {
	Data              []byte `json:"data"`
	CurrentPassphrase string `json:"current_passphrase,omitempty"`

	// Optional. Default values are OK.
	SetDefault    bool   `json:"set_default"`
	NewPassphrase string `json:"new_passphrase"`
}

// Validate validates the import request.
func (i *IdentityImportRequest) Validate() error {
	if len(i.CurrentPassphrase) == 0 {
		return errors.New("current_passphrase must be provided")
	}
	if len(i.Data) == 0 {
		return errors.New("data must be provided")
	}

	return nil
}
