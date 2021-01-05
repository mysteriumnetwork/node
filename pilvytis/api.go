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

package pilvytis

import (
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

// API is object which exposes pilvytis API.
type API struct {
	req    *requests.HTTPClient
	signer identity.SignerFactory
	url    string
}

const (
	orderEndpoint        = "payment/orders"
	currencyEndpoint     = "payment/currencies"
	orderOptionsEndpoint = "payment/order-options"
	exchangeEndpoint     = "payment/exchange-rate"
)

// NewAPI returns a new API instance.
func NewAPI(hc *requests.HTTPClient, url string, signer identity.SignerFactory) *API {
	return &API{
		req:    hc,
		signer: signer,
		url:    url,
	}
}

// OrderStatus is a Coingate order status.
type OrderStatus string

// OrderStatus values.
const (
	OrderStatusNew        OrderStatus = "new"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirming OrderStatus = "confirming"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusInvalid    OrderStatus = "invalid"
	OrderStatusExpired    OrderStatus = "expired"
	OrderStatusCanceled   OrderStatus = "canceled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// Incomplete tells if the order is incomplete and its status needs to be tracked for updates.
func (o OrderStatus) Incomplete() bool {
	switch o {
	case OrderStatusNew, OrderStatusPending, OrderStatusConfirming:
		return true
	}
	return false
}

// OrderResponse is returned from the pilvytis order endpoints.
type OrderResponse struct {
	ID       uint64      `json:"id"`
	Status   OrderStatus `json:"status"`
	Identity string      `json:"identity"`

	MystAmount    float64  `json:"myst_amount"`
	PriceAmount   *float64 `json:"price_amount"`
	PriceCurrency string   `json:"price_currency"`

	PayAmount      *float64 `json:"pay_amount,omitempty"`
	PayCurrency    *string  `json:"pay_currency,omitempty"`
	PaymentAddress string   `json:"payment_address"`
	PaymentURL     string   `json:"payment_url"`

	ReceiveAmount   *float64 `json:"receive_amount"`
	ReceiveCurrency string   `json:"receive_currency"`

	ExpiresAt time.Time `json:"expire_at"`
	CreatedAt time.Time `json:"created_at"`
}

type orderRequest struct {
	MystAmount       float64 `json:"myst_amount"`
	PayCurrency      string  `json:"pay_currency"`
	LightningNetwork bool    `json:"lightning_network"`
}

// CreatePaymentOrder creates a new payment order in the API service.
func (a *API) CreatePaymentOrder(id identity.Identity, mystAmount float64, payCurrency string, lightning bool) (*OrderResponse, error) {
	payload := orderRequest{
		MystAmount:       mystAmount,
		PayCurrency:      payCurrency,
		LightningNetwork: lightning,
	}

	req, err := requests.NewSignedPostRequest(a.url, orderEndpoint, payload, a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	return &resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// GetPaymentOrder returns a payment order by ID from the API
// service that belongs to a given identity.
func (a *API) GetPaymentOrder(id identity.Identity, oid uint64) (*OrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, fmt.Sprintf("%s/%d", orderEndpoint, oid), a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	return &resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// GetPaymentOrders returns a list of payment orders from the API service made by a given identity.
func (a *API) GetPaymentOrders(id identity.Identity) ([]OrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, orderEndpoint, a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp []OrderResponse
	return resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// GetPaymentOrderCurrencies returns a slice of currencies supported for payment orders
func (a *API) GetPaymentOrderCurrencies() ([]string, error) {
	req, err := requests.NewGetRequest(a.url, currencyEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp []string
	return resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// GetPaymentOrderOptions return payment order options
func (a *API) GetPaymentOrderOptions() (*PaymentOrderOptions, error) {
	req, err := requests.NewGetRequest(a.url, orderOptionsEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp PaymentOrderOptions
	return &resp, a.req.DoRequestAndParseResponse(req, &resp)
}

// PaymentOrderOptions represents pilvytis payment order options
type PaymentOrderOptions struct {
	Minimum   float64   `json:"minimum"`
	Suggested []float64 `json:"suggested"`
}

// GetMystExchangeRate returns the exchange rate for myst to other currencies.
func (a *API) GetMystExchangeRate() (map[string]float64, error) {
	req, err := requests.NewGetRequest(a.url, exchangeEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp map[string]float64
	return resp, a.req.DoRequestAndParseResponse(req, &resp)
}
