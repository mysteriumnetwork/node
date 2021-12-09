/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"encoding/json"
	"strings"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
)

// PaymentOrderResponse represents a payment order for mobile usage.
type PaymentOrderResponse struct {
	ID                string          `json:"id"`
	Status            string          `json:"status"`
	IdentityAddress   string          `json:"identity"`
	ChannelAddress    string          `json:"channel_address"`
	Gateway           string          `json:"gateway"`
	ReceiveMYST       string          `json:"receive_myst"`
	PayAmount         string          `json:"pay_amount"`
	PayCurrency       string          `json:"pay_currency"`
	Country           string          `json:"country"`
	Currency          string          `json:"currency"`
	ItemsSubTotal     string          `json:"items_sub_total"`
	TaxRate           string          `json:"tax_rate"`
	TaxSubTotal       string          `json:"tax_sub_total"`
	OrderTotal        string          `json:"order_total"`
	PublicGatewayData json.RawMessage `json:"public_gateway_data"`
}

func newPaymentOrderResponse(r pilvytis.PaymentOrderResponse) PaymentOrderResponse {
	return PaymentOrderResponse{
		ID:                r.ID,
		Status:            r.Status.Status(),
		IdentityAddress:   r.Identity,
		ChannelAddress:    r.ChannelAddress,
		Gateway:           r.GatewayName,
		ReceiveMYST:       r.ReceiveMYST,
		PayAmount:         r.PayAmount,
		PayCurrency:       r.PayCurrency,
		Country:           r.Country,
		Currency:          r.Currency,
		ItemsSubTotal:     r.ItemsSubTotal,
		TaxRate:           r.TaxRate,
		TaxSubTotal:       r.TaxSubTotal,
		OrderTotal:        r.OrderTotal,
		PublicGatewayData: r.PublicGatewayData,
	}
}

// GetPaymentOrderRequest a request to get an order.
type GetPaymentOrderRequest struct {
	IdentityAddress string
	ID              string
}

// GetPaymentGatewayOrder gets an order by ID.
func (mb *MobileNode) GetPaymentGatewayOrder(req *GetPaymentOrderRequest) ([]byte, error) {
	order, err := mb.pilvytis.GetPaymentGatewayOrder(identity.FromAddress(req.IdentityAddress), req.ID)
	if err != nil {
		return nil, err
	}

	res := newPaymentOrderResponse(*order)

	return json.Marshal(res)
}

// GetPaymentGatewayOrderInvoice gets the invoice for an order.
func (mb *MobileNode) GetPaymentGatewayOrderInvoice(req *GetPaymentOrderRequest) ([]byte, error) {
	return mb.pilvytis.GetPaymentGatewayOrderInvoice(identity.FromAddress(req.IdentityAddress), req.ID)
}

// GatewaysResponse represents a respose which cointains gateways and their data.
type GatewaysResponse struct {
	Name         string              `json:"name"`
	OrderOptions PaymentOrderOptions `json:"order_options"`
	Currencies   []string            `json:"currencies"`
}

// PaymentOrderOptions are the suggested and minimum myst amount options for a gateway.
type PaymentOrderOptions struct {
	Minimum   float64   `json:"minimum"`
	Suggested []float64 `json:"suggested"`
}

func newGatewayReponse(g []pilvytis.GatewaysResponse) []GatewaysResponse {
	result := make([]GatewaysResponse, len(g))
	for i, v := range g {
		entry := GatewaysResponse{
			Name: v.Name,
			OrderOptions: PaymentOrderOptions{
				Minimum:   v.OrderOptions.Minimum,
				Suggested: v.OrderOptions.Suggested,
			},
			Currencies: v.Currencies,
		}
		result[i] = entry
	}
	return result
}

// GetGateways returns possible payment gateways.
func (mb *MobileNode) GetGateways() ([]byte, error) {
	gateways, err := mb.pilvytis.GetPaymentGateways()
	if err != nil {
		return nil, err
	}

	return json.Marshal(gateways)
}

// CreatePaymentGatewayOrderReq is used to create a new order.
type CreatePaymentGatewayOrderReq struct {
	IdentityAddress string
	Gateway         string
	MystAmount      string
	PayCurrency     string
	Country         string
	// GatewayCallerData is marshaled json that is accepting by the payment gateway.
	GatewayCallerData []byte
}

// CreatePaymentGatewayOrder creates a payment order.
func (mb *MobileNode) CreatePaymentGatewayOrder(req *CreatePaymentGatewayOrderReq) ([]byte, error) {
	if req.Country == "" {
		org := mb.locationResolver.GetOrigin()
		req.Country = strings.ToUpper(org.Country)
	}

	order, err := mb.pilvytisOrderIssuer.CreatePaymentGatewayOrder(
		identity.FromAddress(req.IdentityAddress),
		req.Gateway,
		req.MystAmount,
		req.PayCurrency,
		req.Country,
		req.GatewayCallerData,
	)
	if err != nil {
		return nil, err
	}

	res := newPaymentOrderResponse(*order)

	return json.Marshal(res)
}

// ListPaymentGatewayOrders lists all payment orders.
func (mb *MobileNode) ListPaymentGatewayOrders(req *ListOrdersRequest) ([]byte, error) {
	orders, err := mb.pilvytis.GetPaymentGatewayOrders(identity.FromAddress(req.IdentityAddress))
	if err != nil {
		return nil, err
	}

	res := make([]PaymentOrderResponse, len(orders))

	for i := range orders {
		orderRes := newPaymentOrderResponse(orders[i])

		res[i] = orderRes
	}

	return json.Marshal(orders)
}
