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
	"strconv"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
)

// CreateOrderRequest a request to create an order.
type CreateOrderRequest struct {
	IdentityAddress string
	MystAmount      float64
	PayCurrency     string
	Lightning       bool
}

// OrderResponse represents a payment order for mobile usage.
type OrderResponse struct {
	ID              int64    `json:"id"`
	IdentityAddress string   `json:"identity_address"`
	Status          string   `json:"status"`
	MystAmount      float64  `json:"myst_amount"`
	PayCurrency     *string  `json:"pay_currency,omitempty"`
	PayAmount       *float64 `json:"pay_amount,omitempty"`
	PaymentAddress  string   `json:"payment_address"`
	PaymentURL      string   `json:"payment_url"`
	ExpiresAt       string   `json:"expire_at"`
	CreatedAt       string   `json:"created_at"`
}

func newOrderResponse(order pilvytis.OrderResponse) (*OrderResponse, error) {
	id, err := shrinkUint64(order.ID)
	if err != nil {
		return nil, err
	}

	response := &OrderResponse{
		ID:              id,
		IdentityAddress: order.Identity,
		Status:          string(order.Status),
		MystAmount:      order.MystAmount,
		PayCurrency:     order.PayCurrency,
		PayAmount:       order.PayAmount,
		PaymentAddress:  order.PaymentAddress,
		PaymentURL:      order.PaymentURL,
		ExpiresAt:       order.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// CreateOrder creates a payment order.
func (mb *MobileNode) CreateOrder(req *CreateOrderRequest) ([]byte, error) {
	order, err := mb.pilvytisOrderIssuer.CreatePaymentOrder(identity.FromAddress(req.IdentityAddress), req.MystAmount, req.PayCurrency, req.Lightning)
	if err != nil {
		return nil, err
	}

	res, err := newOrderResponse(*order)
	if err != nil {
		return nil, err
	}

	return json.Marshal(res)
}

// GetOrderRequest a request to get an order.
type GetOrderRequest struct {
	IdentityAddress string
	ID              int64
}

// GetOrder gets an order by ID.
func (mb *MobileNode) GetOrder(req *GetOrderRequest) ([]byte, error) {
	order, err := mb.pilvytis.GetPaymentOrder(identity.FromAddress(req.IdentityAddress), uint64(req.ID))
	if err != nil {
		return nil, err
	}

	res, err := newOrderResponse(*order)
	if err != nil {
		return nil, err
	}

	return json.Marshal(res)
}

// ListOrdersRequest a request to list orders.
type ListOrdersRequest struct {
	IdentityAddress string
}

// ListOrders lists all payment orders.
func (mb *MobileNode) ListOrders(req *ListOrdersRequest) ([]byte, error) {
	orders, err := mb.pilvytis.GetPaymentOrders(identity.FromAddress(req.IdentityAddress))
	if err != nil {
		return nil, err
	}

	res := make([]OrderResponse, len(orders))

	for i := range orders {
		orderRes, err := newOrderResponse(orders[i])
		if err != nil {
			return nil, err
		}

		res[i] = *orderRes
	}

	return json.Marshal(orders)
}

// Currencies lists supported payment currencies.
func (mb *MobileNode) Currencies() ([]byte, error) {
	currencies, err := mb.pilvytis.GetPaymentOrderCurrencies()
	if err != nil {
		return nil, err
	}

	return json.Marshal(currencies)
}

// ExchangeRate returns MYST rate in quote currency.
func (mb *MobileNode) ExchangeRate(quote string) (float64, error) {
	return mb.pilvytis.GetMystExchangeRateFor(quote)
}

// OrderUpdatedCallbackPayload is the payload of OrderUpdatedCallback.
type OrderUpdatedCallbackPayload struct {
	OrderID     string
	Status      string
	PayAmount   string
	PayCurrency string
}

// OrderUpdatedCallback is a callback when order status changes.
type OrderUpdatedCallback interface {
	OnUpdate(payload *OrderUpdatedCallbackPayload)
}

// RegisterOrderUpdatedCallback registers OrderStatusChanged callback.
func (mb *MobileNode) RegisterOrderUpdatedCallback(cb OrderUpdatedCallback) {
	_ = mb.eventBus.SubscribeAsync(pilvytis.AppTopicOrderUpdated, func(e pilvytis.AppEventOrderUpdated) {
		payload := OrderUpdatedCallbackPayload{}
		payload.OrderID = e.ID
		payload.Status = e.Status.Status()
		payload.PayAmount = e.PayAmount
		payload.PayCurrency = e.PayCurrency
		cb.OnUpdate(&payload)
	})
}

func shrinkUint64(u uint64) (int64, error) {
	return strconv.ParseInt(strconv.FormatUint(u, 10), 10, 64)
}
