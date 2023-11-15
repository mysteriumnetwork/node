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
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/exchange"
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

const exchangeEndpoint = "api/v1/payment/exchange-rate"

type addressProvider interface {
	GetActiveChannelAddress(chainID int64, id common.Address) (common.Address, error)
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
func (a *API) GetPaymentGateways(optionsCurrency exchange.Currency) ([]GatewaysResponse, error) {
	query := url.Values{}
	query.Set("options_currency", string(optionsCurrency))
	req, err := requests.NewGetRequest(a.url, "api/v2/payment/gateways", query)
	if err != nil {
		return nil, err
	}

	var resp []GatewaysResponse
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GatewayOrderResponse is a response for a payment order.
type GatewayOrderResponse struct {
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
	State   string `json:"state"`

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
func (a *API) GetPaymentGatewayOrders(id identity.Identity) ([]GatewayOrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, "api/v2/payment/orders", a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp []GatewayOrderResponse
	return resp, a.sendRequestAndParseResp(req, &resp)
}

// GetPaymentGatewayOrder returns a payment order by ID from the API
// service that belongs to a given identity.
func (a *API) GetPaymentGatewayOrder(id identity.Identity, oid string) (*GatewayOrderResponse, error) {
	req, err := requests.NewSignedGetRequest(a.url, fmt.Sprintf("api/v2/payment/orders/%s", oid), a.signer(id))
	if err != nil {
		return nil, err
	}

	var resp GatewayOrderResponse
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
	return io.ReadAll(res.Body)
}

// GatewayClientCallback triggers a payment callback from the client-side.
// We will query the payment provider to verify the payment.
func (a *API) GatewayClientCallback(id identity.Identity, gateway string, payload any) error {
	req, err := requests.NewSignedPostRequest(a.url, fmt.Sprintf("api/v2/payment/%s/client-callback", gateway), payload, a.signer(id))
	if err != nil {
		return err
	}
	var resp struct{}
	return a.sendRequestAndParseResp(req, &resp)
}

type paymentOrderRequest struct {
	ChannelAddress string `json:"channel_address"`
	MystAmount     string `json:"myst_amount"`
	AmountUSD      string `json:"amount_usd"`
	PayCurrency    string `json:"pay_currency"`
	Country        string `json:"country"`
	State          string `json:"state"`
	ChainID        int64  `json:"chain_id"`
	ProjectId      string `json:"project_id"`

	GatewayCallerData json.RawMessage `json:"gateway_caller_data"`
}

// GatewayOrderRequest for creating payment gateway order
type GatewayOrderRequest struct {
	Identity    identity.Identity
	Gateway     string
	MystAmount  string
	AmountUSD   string
	PayCurrency string
	Country     string
	State       string
	ProjectID   string
	CallerData  json.RawMessage
}

// createPaymentOrder creates a new payment order in the API service.
func (a *API) createPaymentGatewayOrder(cgo GatewayOrderRequest) (*GatewayOrderResponse, error) {
	chainID := config.Current.GetInt64(config.FlagChainID.Name)

	ch, err := a.channelCalculator.GetActiveChannelAddress(chainID, cgo.Identity.ToCommonAddress())
	if err != nil {
		return nil, fmt.Errorf("could get channel address: %w", err)
	}
	//https: //sandbox-pilvytis.mysterium.network
	payload := paymentOrderRequest{
		ChannelAddress:    ch.Hex(),
		MystAmount:        cgo.MystAmount,
		AmountUSD:         cgo.AmountUSD,
		PayCurrency:       cgo.PayCurrency,
		Country:           cgo.Country,
		State:             cgo.State,
		ChainID:           chainID,
		GatewayCallerData: cgo.CallerData,
		ProjectId:         cgo.ProjectID,
	}

	path := fmt.Sprintf("api/v2/payment/%s/orders", cgo.Gateway)
	req, err := requests.NewSignedPostRequest(a.url, path, payload, a.signer(cgo.Identity))
	if err != nil {
		return nil, err
	}

	var resp GatewayOrderResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}

func (a *API) sendRequestAndParseResp(req *http.Request, resp interface{}) error {
	loc := a.lp.GetOrigin()

	req.Header.Set("X-Origin-Country", loc.Country)
	req.Header.Set("X-Origin-OS", runtime.GOOS)
	req.Header.Set("X-Origin-Node-Version", metadata.VersionAsString())

	return a.req.DoRequestAndParseResponse(req, &resp)
}

// RegistrationPaymentResponse is a response for the status of a registration payment.
type RegistrationPaymentResponse struct {
	Paid bool `json:"paid"`
}

// GetRegistrationPaymentStatus returns whether a registration payment order
// has been paid by a given identity
func (a *API) GetRegistrationPaymentStatus(id identity.Identity) (*RegistrationPaymentResponse, error) {
	req, err := requests.NewGetRequest(a.url, fmt.Sprintf("api/v2/payment/registration/%s", id.Address), nil)
	if err != nil {
		return nil, err
	}

	var resp RegistrationPaymentResponse
	return &resp, a.sendRequestAndParseResp(req, &resp)
}
