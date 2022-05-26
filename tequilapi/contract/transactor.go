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
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-openapi/strfmt"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// FeesDTO represents the transactor fees
// swagger:model FeesDTO
type FeesDTO struct {
	Registration       *big.Int `json:"registration"`
	RegistrationTokens Tokens   `json:"registration_tokens"`
	Settlement         *big.Int `json:"settlement"`
	SettlementTokens   Tokens   `json:"settlement_tokens"`
	// deprecated - confusing name
	Hermes              uint16   `json:"hermes"`
	HermesPercent       string   `json:"hermes_percent"`
	DecreaseStake       *big.Int `json:"decreaseStake"`
	DecreaseStakeTokens Tokens   `json:"decrease_stake_tokens"`
}

// CombinedFeesResponse represents transactor fees.
// swagger:model CombinedFeesResponse
type CombinedFeesResponse struct {
	Current TransactorFees `json:"current"`
	Last    TransactorFees `json:"last"`

	ServerTime    time.Time `json:"server_time"`
	HermesPercent string    `json:"hermes_percent"`
}

// TransactorFees represents transactor fees.
// swagger:model TransactorFees
type TransactorFees struct {
	Registration  Tokens `json:"registration"`
	Settlement    Tokens `json:"settlement"`
	DecreaseStake Tokens `json:"decrease_stake"`

	ValidUntil time.Time `json:"valid_until"`
}

// NewTransactorFees converts registry fees to public api.
func NewTransactorFees(r *registry.Fees) TransactorFees {
	return TransactorFees{
		Registration:  NewTokens(r.Register),
		Settlement:    NewTokens(r.Settle),
		DecreaseStake: NewTokens(r.DecreaseStake),
		ValidUntil:    r.ValidUntil,
	}
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
func (q *SettlementListQuery) Bind(request *http.Request) *apierror.APIError {
	v := apierror.NewValidator()
	if err := q.PaginationQuery.Bind(request); err != nil {
		for field, fieldErr := range err.Err.Fields {
			v.Fail(field, fieldErr.Code, fieldErr.Message)
		}
	}

	qs := request.URL.Query()
	if qStr := qs.Get("date_from"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			v.Invalid("date_from", "Could not parse 'date_from'")
		} else {
			q.DateFrom = qVal
		}
	}
	if qStr := qs.Get("date_to"); qStr != "" {
		if qVal, err := parseDate(qStr); err != nil {
			v.Invalid("date_to", "Could not parse 'date_to'")
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
		q.Types = append(q.Types, types...)
	}

	return v.Err()
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
	HermesIDs  []common.Address `json:"hermes_ids"`
	ProviderID string           `json:"provider_id"`

	// Deprecated
	HermesID string `json:"hermes_id"`
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
func (w *WithdrawRequest) Validate() *apierror.APIError {
	v := apierror.NewValidator()
	zeroAddr := common.HexToAddress("").Hex()
	if !common.IsHexAddress(w.HermesID) || w.HermesID == zeroAddr {
		v.Invalid("hermes_id", "'hermes_id' should be a valid hex address")
	}
	if !common.IsHexAddress(w.ProviderID) || w.ProviderID == zeroAddr {
		v.Invalid("provider_id", "'provider_id' should be a valid hex address")
	}
	if !common.IsHexAddress(w.Beneficiary) || w.Beneficiary == zeroAddr {
		v.Invalid("beneficiary", "'beneficiary' should be a valid hex address")
	}

	amount, err := w.AmountInMYST()
	if err != nil {
		v.Invalid("amount", err.Error())
	} else if amount != nil && amount.Cmp(crypto.FloatToBigMyst(99)) > 0 {
		v.Invalid("amount", "withdrawal amount cannot be more than 99 MYST")
	}

	return v.Err()
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
	ProviderID  string `json:"provider_id"`
	HermesID    string `json:"hermes_id"`
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
