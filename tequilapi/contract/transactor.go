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
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/payments/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-openapi/strfmt"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// FeesDTO represents the transactor fees
// swagger:model FeesDTO
type FeesDTO struct {
	Registration  *big.Int `json:"registration"`
	Settlement    *big.Int `json:"settlement"`
	Hermes        uint16   `json:"hermes"`
	DecreaseStake *big.Int `json:"decreaseStake"`
}

// NewSettlementListQuery creates settlement list query with default values.
func NewSettlementListQuery() SettlementListQuery {
	return SettlementListQuery{
		PaginationQuery: NewPaginationQuery(),
	}
}

// SettlementListQuery allows to filter requested settlements.
// swagger:parameters settlementList
type SettlementListQuery struct {
	PaginationQuery

	// Filter the settlements from this date. Formatted in RFC3339 e.g. 2020-07-01.
	// in: query
	DateFrom *strfmt.Date `json:"date_from"`

	// Filter the settlements until this date Formatted in RFC3339 e.g. 2020-07-30.
	// in: query
	DateTo *strfmt.Date `json:"date_to"`

	// Provider identity to filter the sessions by.
	// in: query
	ProviderID *string `json:"provider_id"`

	// Hermes ID to filter the sessions by.
	// in: query
	HermesID *string `json:"hermes_id"`

	// Settlement type to filter the sessions by. "settlement" or "withdrawal"
	// in: query
	Types []string `json:"types"`
}

// Bind creates and validates query from API request.
func (q *SettlementListQuery) Bind(request *http.Request) *validation.FieldErrorMap {
	errs := validation.NewErrorMap()
	errs.Set(q.PaginationQuery.Bind(request))

	qs := request.URL.Query()
	if qStr := qs.Get("date_from"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			errs.ForField("date_from").Add(err)
		} else {
			q.DateFrom = qVal
		}
	}
	if qStr := qs.Get("date_to"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			errs.ForField("date_to").Add(err)
		} else {
			q.DateTo = qVal
		}
	}
	if qStr := qs.Get("provider_id"); qStr != "" {
		q.ProviderID = &qStr
	}
	if qStr := qs.Get("hermes_id"); qStr != "" {
		q.HermesID = &qStr
	}

	if types, ok := qs["types"]; ok {
		for _, sv := range types {
			q.Types = append(q.Types, sv)
		}
	}

	return errs
}

// ToFilter converts API query to storage filter.
func (q *SettlementListQuery) ToFilter() pingpong.SettlementHistoryFilter {
	filter := pingpong.SettlementHistoryFilter{}
	if q.DateFrom != nil {
		timeFrom := time.Time(*q.DateFrom).Truncate(24 * time.Hour)
		filter.TimeFrom = &timeFrom
	}
	if q.DateTo != nil {
		timeTo := time.Time(*q.DateTo).Truncate(24 * time.Hour).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
		filter.TimeTo = &timeTo
	}
	if q.ProviderID != nil {
		providerID := identity.FromAddress(*q.ProviderID)
		filter.ProviderID = &providerID
	}
	if q.HermesID != nil {
		hermesID := common.HexToAddress(*q.HermesID)
		filter.HermesID = &hermesID
	}
	if q.Types != nil {
		for _, qt := range q.Types {
			filter.Types = append(filter.Types, pingpong.HistoryType(qt))
		}
	}
	return filter
}

// NewSettlementListResponse maps to API settlement list.
func NewSettlementListResponse(
	WithdrawalTotal *big.Int,
	settlements []pingpong.SettlementHistoryEntry,
	paginator *utils.Paginator,
) SettlementListResponse {
	dtoArray := make([]SettlementDTO, len(settlements))
	for i, settlement := range settlements {
		dtoArray[i] = NewSettlementDTO(settlement)
	}

	return SettlementListResponse{
		Items:           dtoArray,
		PageableDTO:     NewPageableDTO(paginator),
		WithdrawalTotal: WithdrawalTotal.String(),
	}
}

// SettlementListResponse defines settlement list representable as json.
// swagger:model SettlementListResponse
type SettlementListResponse struct {
	Items           []SettlementDTO `json:"items"`
	WithdrawalTotal string          `json:"withdrawal_total"`
	PageableDTO
}

// NewSettlementDTO maps to API settlement.
func NewSettlementDTO(settlement pingpong.SettlementHistoryEntry) SettlementDTO {
	return SettlementDTO{
		TxHash:           settlement.TxHash.Hex(),
		ProviderID:       settlement.ProviderID.Address,
		HermesID:         settlement.HermesID.Hex(),
		ChannelAddress:   settlement.ChannelAddress.Hex(),
		Beneficiary:      settlement.Beneficiary.Hex(),
		Amount:           settlement.Amount,
		SettledAt:        settlement.Time.Format(time.RFC3339),
		Fees:             settlement.Fees,
		Error:            settlement.Error,
		IsWithdrawal:     settlement.IsWithdrawal,
		BlockExplorerURL: settlement.BlockExplorerURL,
	}
}

// SettlementDTO represents the settlement object.
// swagger:model SettlementDTO
type SettlementDTO struct {
	// example: 0x20c070a9be65355adbd2ba479e095e2e8ed7e692596548734984eab75d3fdfa5
	TxHash string `json:"tx_hash"`

	// example: 0x0000000000000000000000000000000000000001
	ProviderID string `json:"provider_id"`

	// example: 0x0000000000000000000000000000000000000001
	HermesID string `json:"hermes_id"`

	// example: 0x0000000000000000000000000000000000000001
	ChannelAddress string `json:"channel_address"`

	// example: 0x0000000000000000000000000000000000000001
	Beneficiary string `json:"beneficiary"`

	// example: 500000
	Amount *big.Int `json:"amount"`

	// example: 2019-06-06T11:04:43.910035Z
	SettledAt string `json:"settled_at"`

	// example: 500000
	Fees *big.Int `json:"fees"`

	// example: false
	IsWithdrawal bool `json:"is_withdrawal"`

	// example: https://example.com
	BlockExplorerURL string `json:"block_explorer_url"`

	// example: internal server error
	Error string `json:"error"`
}

// SettleRequest represents the request to settle hermes promises
// swagger:model SettleRequestDTO
type SettleRequest struct {
	HermesID   string `json:"hermes_id"`
	ProviderID string `json:"provider_id"`
}

// WithdrawRequest represents the request to withdraw earnings to l1.
// swagger:model WithdrawRequestDTO
type WithdrawRequest struct {
	HermesID    string `json:"hermes_id"`
	ProviderID  string `json:"provider_id"`
	Beneficiary string `json:"beneficiary"`
	FromChainID int64  `json:"from_chain_id"`
	ToChainID   int64  `json:"to_chain_id"`
	Amount      string `json:"amount,omitempty"`
}

// Validate will validate a given request
func (w *WithdrawRequest) Validate() error {
	zeroAddr := common.HexToAddress("").Hex()
	if !common.IsHexAddress(w.HermesID) || w.HermesID == zeroAddr {
		return errors.New("hermesID should be a valid hex address")
	}
	if !common.IsHexAddress(w.ProviderID) || w.ProviderID == zeroAddr {
		return errors.New("providerID should be a valid hex address")
	}
	if !common.IsHexAddress(w.Beneficiary) || w.Beneficiary == zeroAddr {
		return errors.New("beneficiary should be a valid hex address")
	}

	amount, err := w.AmountInMYST()
	if err != nil {
		return err
	}

	if amount != nil && amount.Cmp(crypto.FloatToBigMyst(99)) > 0 {
		return errors.New("withdrawal amount cannot be more than 99 MYST")
	}

	return nil
}

// AmountInMYST will return the amount value converted to big.Int MYST.
//
// Amount can be `nil`
func (w *WithdrawRequest) AmountInMYST() (*big.Int, error) {
	if w.Amount == "" {
		return nil, nil
	}

	res, ok := big.NewInt(0).SetString(w.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("%v is not a valid integer", w.Amount)
	}

	return res, nil
}

// SettleWithBeneficiaryRequest represent the request to settle with new beneficiary address.
type SettleWithBeneficiaryRequest struct {
	SettleRequest
	Beneficiary string `json:"beneficiary"`
}

// DecreaseStakeRequest represents the decrease stake request
// swagger:model DecreaseStakeRequest
type DecreaseStakeRequest struct {
	ID     string   `json:"id,omitempty"`
	Amount *big.Int `json:"amount,omitempty"`
}

// ReferralTokenResponse represents a response for referral token.
// swagger:model ReferralTokenResponse
type ReferralTokenResponse struct {
	Token string `json:"token"`
}

// BeneficiaryTxStatus settle with beneficiary transaction status.
// swagger:model BeneficiaryTxStatus
type BeneficiaryTxStatus struct {
	State beneficiary.SettleState `json:"state"`
	Error string                  `json:"error"`
	// example: 0x0000000000000000000000000000000000000001
	ChangeTo string `json:"change_to"`
}

// TokenRewardAmount represents a response for token rewards.
// swagger:model TokenRewardAmount
type TokenRewardAmount struct {
	Amount *big.Int `json:"amount"`
}

// ChainSummary represents a response for token rewards.
// swagger:model ChainSummary
type ChainSummary struct {
	Chains       map[int64]string `json:"chains"`
	CurrentChain int64            `json:"current_chain"`
}
