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

// IdentityDTO holds identity information.
// swagger:model IdentityDTO
type IdentityDTO struct {
	// identity in Ethereum address format
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	Address            string `json:"id"`
	RegistrationStatus string `json:"registration_status"`
	ChannelAddress     string `json:"channel_address"`
	Balance            uint64 `json:"balance"`
	Earnings           uint64 `json:"earnings"`
	EarningsTotal      uint64 `json:"earnings_total"`
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

// IdentityRequest request used for identity actions.
// swagger:model IdentityRequestDTO
type IdentityRequest struct {
	Address    *string `json:"id"`
	Passphrase *string `json:"passphrase"`
}

// ValidateIdentityRequest validates request.
func ValidateIdentityRequest(req IdentityRequest) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	if req.Passphrase == nil {
		errors.ForField("passphrase").AddError("required", "Field is required")
	}
	return
}

// IdentityRegistrationResponse represents registration status and needed data for registering of given identity
// swagger:model RegistrationDataDTO
type IdentityRegistrationResponse struct {
	Status string `json:"status"`
	// Returns true if identity is registered in payments smart contract
	Registered bool `json:"registered"`
}

// ListProposalsResponse holds list of proposals.
// swagger:model ListProposalsResponse
type ListProposalsResponse struct {
	Proposals []*ProposalDTO `json:"proposals"`
}
