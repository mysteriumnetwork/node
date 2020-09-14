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
	"time"

	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/vcraescu/go-paginator"
)

// FeesDTO represents the transactor fees
// swagger:model FeesDTO
type FeesDTO struct {
	Registration  *big.Int `json:"registration"`
	Settlement    *big.Int `json:"settlement"`
	Hermes        uint16   `json:"hermes"`
	DecreaseStake *big.Int `json:"decreaseStake"`
}

// NewSettlementListResponse maps to API settlement list.
func NewSettlementListResponse(
	settlements []pingpong.SettlementHistoryEntry,
	paginator *paginator.Paginator,
) ListSettlementsResponse {
	dtoArray := make([]SettlementDTO, len(settlements))
	for i, settlement := range settlements {
		dtoArray[i] = NewSettlementDTO(settlement)
	}

	return ListSettlementsResponse{
		Settlements: dtoArray,
		Paging:      NewPagingDTO(paginator),
	}
}

// ListSettlementsResponse defines settlement list representable as json.
// swagger:model ListSettlementsResponse
type ListSettlementsResponse struct {
	Settlements []SettlementDTO `json:"settlements"`
	Paging      PagingDTO       `json:"paging"`
}

// NewSettlementDTO maps to API settlement.
func NewSettlementDTO(settlement pingpong.SettlementHistoryEntry) SettlementDTO {
	return SettlementDTO{
		TxHash:         settlement.TxHash.Hex(),
		ProviderID:     settlement.ProviderID.Address,
		HermesID:       settlement.HermesID.Hex(),
		ChannelAddress: settlement.ChannelAddress.Hex(),
		Beneficiary:    settlement.Beneficiary.Hex(),
		Amount:         settlement.Amount.Uint64(),
		SettledAt:      settlement.Time.Format(time.RFC3339),
	}
}

// SettlementDTO represents the settlement object.
// swagger:model SettlementDTO
type SettlementDTO struct {
	// example: 0x20c070a9be65355adbd2ba479e095e2e8ed7e692596548734984eab75d3fdfa5
	TxHash string `json:"tx_hash" storm:"id"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// example: 0x0000000000000000000000000000000000000001
	HermesID string `json:"hermes_id"`

	// example: 0x0000000000000000000000000000000000000001
	ChannelAddress string `json:"channel_address"`

	// example: 0x0000000000000000000000000000000000000001
	Beneficiary string `json:"beneficiary"`

	// example: 500000
	Amount uint64 `json:"amount"`

	// example: 2019-06-06T11:04:43.910035Z
	SettledAt string `json:"settled_at"`
}

// SettleRequest represents the request to settle hermes promises
// swagger:model SettleRequestDTO
type SettleRequest struct {
	HermesID   string `json:"hermes_id"`
	ProviderID string `json:"provider_id"`
}

// SettleWithBeneficiaryRequest represent the request to settle with new beneficiary address.
type SettleWithBeneficiaryRequest struct {
	SettleRequest
	Beneficiary string `json:"beneficiary"`
}

// DecreaseStakeRequest represents the decrease stake request
// swagger:model DecreaseStakeRequest
type DecreaseStakeRequest struct {
	ID            string   `json:"id,omitempty"`
	Amount        *big.Int `json:"amount,omitempty"`
	TransactorFee *big.Int `json:"transactor_fee,omitempty"`
}
