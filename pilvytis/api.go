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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/requests"
)

// API is object which exposes pilvytis API.
type API struct {
	req               *requests.HTTPClient
	channelCalculator addressProvider
	signer            identity.SignerFactory
	lp                locationProvider
	url               string
}

type locationProvider interface {
	GetOrigin() locationstate.Location
}

const (
	orderEndpoint        = "api/v1/payment/orders"
	currencyEndpoint     = "api/v1/payment/currencies"
	orderOptionsEndpoint = "api/v1/payment/order-options"
	exchangeEndpoint     = "api/v1/payment/exchange-rate"
)

type addressProvider interface {
	GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error)
}

// NewAPI returns a new API instance.
func NewAPI(hc *requests.HTTPClient, url string, signer identity.SignerFactory, lp locationProvider, cc addressProvider) *API {
	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}

	return &API{
		req:               hc,
		signer:            signer,
		url:               url,
		lp:                lp,
		channelCalculator: cc,
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

// Status returns a current status.
// Its part of StatusProvider interface.
func (o OrderStatus) Status() string {
	return string(o)
}

// Paid tells if the order has been paid for.
func (o OrderStatus) Paid() bool {
	return o == OrderStatusPaid
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
	ChannelAddress   string  `json:"channel_address"`
	MystAmount       float64 `json:"myst_amount"`
	PayCurrency      string  `json:"pay_currency"`
	LightningNetwork bool    `json:"lightning_network"`
	ChainID          int64   `json:"chain_id"`
}

// createPaymentOrder creates a new payment order in the API service.
func (a *API) createPaymentOrder(id identity.Identity, mystAmount float64, payCurrency string, lightning bool) (*OrderResponse, error) {
	chainID := config.Current.GetInt64(config.FlagChainID.Name)

	ch, err := a.channelCalculator.GetChannelAddress(chainID, id)
	if err != nil {
		return nil, fmt.Errorf("could get channel address: %w", err)
	}

	payload := orderRequest{
		ChannelAddress:   ch.Hex(),
		MystAmount:       mystAmount,
		PayCurrency:      payCurrency,
		LightningNetwork: lightning,
		ChainID:          chainID,
	}

	req, err := requests.NewSignedPostRequest(a.url, orderEndpoint, payload, a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentOrder returns a payment order by ID from the API
// service that belongs to a given identity.
func (a *API) GetPaymentOrder(id identity.Identity, oid uint64) (*OrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, fmt.Sprintf("%s/%d", orderEndpoint, oid), a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp OrderResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentOrders returns a list of payment orders from the API service made by a given identity.
func (a *API) GetPaymentOrders(id identity.Identity) ([]OrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, orderEndpoint, a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp []OrderResponse
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentOrderCurrencies returns a slice of currencies supported for payment orders
func (a *API) GetPaymentOrderCurrencies() ([]string, error) {
	req, err := requests.NewGetRequest(a.url, currencyEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp []string
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentOrderOptions return payment order options
func (a *API) GetPaymentOrderOptions() (*PaymentOrderOptions, error) {
	req, err := requests.NewGetRequest(a.url, orderOptionsEndpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp PaymentOrderOptions
	return &resp, a.sendRequestAndParseResp(req, &resp)
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
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GetMystExchangeRateFor returns the exchange rate for myst to for a given currency currencies.
func (a *API) GetMystExchangeRateFor(curr string) (float64, error) {
	rates, err := a.GetMystExchangeRate()
	if err != nil {
		return 0, err
	}
	rate, ok := rates[strings.ToUpper(curr)]
	if !ok {
		return 0, errors.New("currency not supported")
	}
	return rate, nil
}

// GatewaysResponse holds data about payment gateways.
type GatewaysResponse struct {
	Name         string              `json:"name"`
	OrderOptions PaymentOrderOptions `json:"order_options"`
	Currencies   []string            `json:"currencies"`
}

// GetPaymentGateways returns a slice of supported gateways.
func (a *API) GetPaymentGateways() ([]GatewaysResponse, error) {
	req, err := requests.NewGetRequest(a.url, "api/v2/payment/gateways", nil)
	if err != nil {
		return nil, err
	}

	var resp []GatewaysResponse
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// PaymentOrderResponse is a response for a payment order.
type PaymentOrderResponse struct {
	ID     string             `json:"id"`
	Status PaymentOrderStatus `json:"status"`

	Identity       string `json:"identity"`
	ChainID        int64  `json:"chain_id"`
	ChannelAddress string `json:"channel_address"`

	GatewayName string `json:"gateway_name"`

	ReceiveMYST string `json:"receive_myst"`
	PayAmount   string `json:"pay_amount"`
	PayCurrency string `json:"pay_currency"`

	Country string `json:"country"`

	Currency      string `json:"currency"`
	ItemsSubTotal string `json:"items_sub_total"`
	TaxRate       string `json:"tax_rate"`
	TaxSubTotal   string `json:"tax_sub_total"`
	OrderTotal    string `json:"order_total"`

	PublicGatewayData json.RawMessage `json:"public_gateway_data"`
}

// PaymentOrderStatus defines a status for a payment order.
type PaymentOrderStatus string

const (
	// PaymentOrderStatusInitial defines a status for any payment orders not sent to the
	// payment service provider.
	PaymentOrderStatusInitial PaymentOrderStatus = "initial"
	// PaymentOrderStatusNew defines a status for any payment orders sent to the
	// payment service provider.
	PaymentOrderStatusNew PaymentOrderStatus = "new"
	// PaymentOrderStatusPaid defines a status for any payment orders paid and processed by the
	// payment service provider.
	PaymentOrderStatusPaid PaymentOrderStatus = "paid"
	// PaymentOrderStatusFailed defines a status for any payment orders sent to the
	// payment service provider which have failed.
	PaymentOrderStatusFailed PaymentOrderStatus = "failed"
)

// Incomplete tells if the order is incomplete and its status needs to be tracked for updates.
func (p PaymentOrderStatus) Incomplete() bool {
	switch p {
	case PaymentOrderStatusPaid, PaymentOrderStatusFailed:
		return false
	default:
		return true
	}
}

// Paid tells if the order has been paid for.
func (p PaymentOrderStatus) Paid() bool {
	return p == PaymentOrderStatusPaid
}

// Status returns a current status.
// Its part of StatusProvider interface.
func (p PaymentOrderStatus) Status() string {
	return string(p)
}

// GetPaymentGatewayOrders returns a list of payment orders from the API service made by a given identity.
func (a *API) GetPaymentGatewayOrders(id identity.Identity) ([]PaymentOrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, "api/v2/payment/orders", a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp []PaymentOrderResponse
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentGatewayOrder returns a payment order by ID from the API
// service that belongs to a given identity.
func (a *API) GetPaymentGatewayOrder(id identity.Identity, oid string) (*PaymentOrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, fmt.Sprintf("api/v2/payment/orders/%s", oid), a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp PaymentOrderResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentGatewayOrderInvoice returns an invoice for a payment order by ID from the API
// service that belongs to a given identity.
func (a *API) GetPaymentGatewayOrderInvoice(id identity.Identity, oid string) ([]byte, error) {
	req, err := requests.NewSignedGetRequest(a.url, fmt.Sprintf("api/v2/payment/orders/%s/invoice", oid), a.signer(id))
	if err != nil {
		return nil, err
	}

	res, err := a.req.Do(req)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(res.Body)
}

type paymentOrderRequest struct {
	ChannelAddress string `json:"channel_address"`
	MystAmount     string `json:"myst_amount"`
	PayCurrency    string `json:"pay_currency"`
	Country        string `json:"country"`
	ChainID        int64  `json:"chain_id"`

	GatewayCallerData json.RawMessage `json:"gateway_caller_data"`
}

// createPaymentOrder creates a new payment order in the API service.
func (a *API) createPaymentGatewayOrder(id identity.Identity, gateway string, mystAmount string, payCurrency string, country string, callerData json.RawMessage) (*PaymentOrderResponse, error) {
	chainID := config.Current.GetInt64(config.FlagChainID.Name)

	ch, err := a.channelCalculator.GetChannelAddress(chainID, id)
	if err != nil {
		return nil, fmt.Errorf("could get channel address: %w", err)
	}

	payload := paymentOrderRequest{
		ChannelAddress:    ch.Hex(),
		MystAmount:        mystAmount,
		PayCurrency:       payCurrency,
		Country:           country,
		ChainID:           chainID,
		GatewayCallerData: callerData,
	}

	path := fmt.Sprintf("api/v2/payment/%s/orders", gateway)
	req, err := requests.NewSignedPostRequest(a.url, path, payload, a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp PaymentOrderResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}

func (a *API) sendRequestAndParseResp(req *http.Request, resp interface{}) error {
	loc := a.lp.GetOrigin()

	req.Header.Set("X-Origin-Country", loc.Country)
	req.Header.Set("X-Origin-OS", runtime.GOOS)
	req.Header.Set("X-Origin-Node-Version", metadata.VersionAsString())

	return a.req.DoRequestAndParseResponse(req, &resp)
}
