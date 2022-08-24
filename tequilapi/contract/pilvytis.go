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
	"encoding/json"

	"github.com/mysteriumnetwork/node/identity"

	"github.com/mysteriumnetwork/node/pilvytis"
)

// PaymentOrderOptions represents pilvytis payment order options
// swagger:model PaymentOrderOptions
type PaymentOrderOptions struct {
	Minimum   float64   `json:"minimum"`
	Suggested []float64 `json:"suggested"`
}

// ToPaymentOrderOptions - convert pilvytis.PaymentOrderOptions to contract.ToPaymentOrderOptions
func ToPaymentOrderOptions(poo *pilvytis.PaymentOrderOptions) *PaymentOrderOptions {
	return &PaymentOrderOptions{
		Minimum:   poo.Minimum,
		Suggested: poo.Suggested,
	}
}

// PaymentOrderResponse holds payment gateway order details.
// swagger:model PaymentOrderResponse
type PaymentOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`

	Identity       string `json:"identity"`
	ChainID        int64  `json:"chain_id"`
	ChannelAddress string `json:"channel_address"`

	GatewayName string `json:"gateway_name"`

	ReceiveMYST string `json:"receive_myst"`
	PayAmount   string `json:"pay_amount"`
	PayCurrency string `json:"pay_currency"`
	Country     string `json:"country"`
	State       string `json:"state"`

	Currency      string `json:"currency"`
	ItemsSubTotal string `json:"items_sub_total"`
	TaxRate       string `json:"tax_rate"`
	TaxSubTotal   string `json:"tax_sub_total"`
	OrderTotal    string `json:"order_total"`

	PublicGatewayData json.RawMessage `json:"public_gateway_data"`
}

// NewPaymentOrderResponse creates an instance of PaymentOrderResponse
func NewPaymentOrderResponse(r *pilvytis.GatewayOrderResponse) PaymentOrderResponse {
	return PaymentOrderResponse{
		ID:                r.ID,
		Status:            string(r.Status),
		Identity:          r.Identity,
		ChainID:           r.ChainID,
		ChannelAddress:    r.ChannelAddress,
		GatewayName:       r.GatewayName,
		ReceiveMYST:       r.ReceiveMYST,
		PayAmount:         r.PayAmount,
		PayCurrency:       r.PayCurrency,
		Country:           r.Country,
		State:             r.State,
		Currency:          r.Currency,
		ItemsSubTotal:     r.ItemsSubTotal,
		TaxRate:           r.TaxRate,
		TaxSubTotal:       r.TaxSubTotal,
		OrderTotal:        r.OrderTotal,
		PublicGatewayData: r.PublicGatewayData,
	}
}

// NewPaymentOrdersResponse creates a slice of orders response
func NewPaymentOrdersResponse(r []pilvytis.GatewayOrderResponse) []PaymentOrderResponse {
	result := make([]PaymentOrderResponse, len(r))
	for i := range r {
		result[i] = NewPaymentOrderResponse(&r[i])
	}
	return result
}

// RegistrationPaymentResponse holds a registration payment order response.
// swagger:model RegistrationPaymentResponse
type RegistrationPaymentResponse struct {
	Paid bool `json:"paid"`
}

// NewRegistrationPaymentResponse creates a registration order response
func NewRegistrationPaymentResponse(r *pilvytis.RegistrationPaymentResponse) RegistrationPaymentResponse {
	return RegistrationPaymentResponse{
		Paid: r.Paid,
	}
}

// GatewaysResponse holds payment gateway details.
// swagger:model GatewaysResponse
type GatewaysResponse struct {
	Name         string              `json:"name"`
	OrderOptions PaymentOrderOptions `json:"order_options"`
	Currencies   []string            `json:"currencies"`
}

// ToGatewaysReponse converts a pilvytis gateway response to contract.
func ToGatewaysReponse(g []pilvytis.GatewaysResponse) []GatewaysResponse {
	result := make([]GatewaysResponse, len(g))
	for i, v := range g {
		entry := GatewaysResponse{
			Name:         v.Name,
			OrderOptions: *ToPaymentOrderOptions(&v.OrderOptions),
			Currencies:   v.Currencies,
		}
		result[i] = entry
	}
	return result
}

// PaymentOrderRequest holds order request details
// swagger:model PaymentOrderRequest
type PaymentOrderRequest struct {
	// example: 3.14
	MystAmount string `json:"myst_amount"`

	// example: 20
	AmountUSD string `json:"amount_usd"`

	// example: EUR
	PayCurrency string `json:"pay_currency"`

	// example: US
	Country string `json:"country"`

	// example: MO
	State string `json:"state"`

	// example: mysteriumvpn, mystnodes
	ProjectID string `json:"project_id"`

	// example: {}
	CallerData json.RawMessage `json:"gateway_caller_data"`
}

// GatewayOrderRequest convenience mapper
func (o *PaymentOrderRequest) GatewayOrderRequest(identity identity.Identity, gateway string) pilvytis.GatewayOrderRequest {
	return pilvytis.GatewayOrderRequest{
		Identity:    identity,
		Gateway:     gateway,
		MystAmount:  o.MystAmount,
		AmountUSD:   o.AmountUSD,
		PayCurrency: o.PayCurrency,
		Country:     o.Country,
		State:       o.State,
		ProjectID:   o.ProjectID,
		CallerData:  o.CallerData,
	}
}
