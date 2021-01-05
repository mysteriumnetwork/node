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

import "github.com/mysteriumnetwork/node/pilvytis"

// OrderRequest holds order request details
// swagger:model OrderRequest
type OrderRequest struct {
	// example: 3.14
	MystAmount float64 `json:"myst_amount"`

	// example: EUR
	PayCurrency string `json:"pay_currency"`

	// example: false
	LightningNetwork bool `json:"lightning_network"`
}

// OrderResponse holds order request details
// swagger:model OrderResponse
type OrderResponse struct {
	// example: 1
	ID uint64 `json:"id"`
	// example: pending
	Status string `json:"status"`
	// example: 0x0000000000000000000000000000000000000002
	Identity string `json:"identity"`

	// example: 1.1
	MystAmount float64 `json:"myst_amount"`
	// example: 1.1
	PriceAmount *float64 `json:"price_amount"`
	// example: BTC
	PriceCurrency string `json:"price_currency"`

	// example: 1.1
	PayAmount *float64 `json:"pay_amount,omitempty"`
	// example: BTC
	PayCurrency *string `json:"pay_currency,omitempty"`
	// example: 0x0000000000000000000000000000000000000002
	PaymentAddress string `json:"payment_address"`
	// example: http://coingate.com/invoice/4949cf0a-fccb-4cc2-9342-7af1890cc664
	PaymentURL string `json:"payment_url"`

	// example: 1.1
	ReceiveAmount *float64 `json:"receive_amount"`
	// example: BTC
	ReceiveCurrency string `json:"receive_currency"`
}

// NewOrderResponse creates a new order response
func NewOrderResponse(r pilvytis.OrderResponse) OrderResponse {
	return OrderResponse{
		ID:              r.ID,
		Status:          string(r.Status),
		Identity:        r.Identity,
		MystAmount:      r.MystAmount,
		PriceAmount:     r.PriceAmount,
		PriceCurrency:   r.PriceCurrency,
		PayAmount:       r.PayAmount,
		PayCurrency:     r.PayCurrency,
		PaymentAddress:  r.PaymentAddress,
		PaymentURL:      r.PaymentURL,
		ReceiveAmount:   r.ReceiveAmount,
		ReceiveCurrency: r.ReceiveCurrency,
	}
}

// NewOrdersResponse creates a slice of orders response
func NewOrdersResponse(r []pilvytis.OrderResponse) []OrderResponse {
	result := make([]OrderResponse, len(r))
	for i := range r {
		result[i] = NewOrderResponse(r[i])
	}
	return result
}
