/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockPilvytis struct {
	currencies []string
	identity   string
	resp       pilvytis.OrderResponse
	respgw     pilvytis.PaymentOrderResponse
}

type mockPilvytisIssuer struct {
	identity string
	resp     pilvytis.OrderResponse
	respgw   pilvytis.PaymentOrderResponse
}

func (mock *mockPilvytisIssuer) CreatePaymentOrder(i identity.Identity, mystAmount float64, payCurrency string, ln bool) (*pilvytis.OrderResponse, error) {
	if i.Address != mock.identity {
		return nil, errors.New("wrong identity")
	}

	return &mock.resp, nil
}

func (mock *mockPilvytisIssuer) CreatePaymentGatewayOrder(id identity.Identity, gw, mystAmount, payCurrency, country string, callerData json.RawMessage) (*pilvytis.PaymentOrderResponse, error) {
	if id.Address != mock.identity {
		return nil, errors.New("wrong identity")
	}

	return &mock.respgw, nil
}

func (mock *mockPilvytis) GetPaymentOrder(i identity.Identity, oid uint64) (*pilvytis.OrderResponse, error) {
	if i.Address != mock.identity {
		return nil, errors.New("wrong identity")
	}

	return &mock.resp, nil
}

func (mock *mockPilvytis) GetPaymentOrders(i identity.Identity) ([]pilvytis.OrderResponse, error) {
	if i.Address != mock.identity {
		return nil, errors.New("wrong identity")
	}

	return []pilvytis.OrderResponse{mock.resp}, nil
}

func (mock *mockPilvytis) GetPaymentOrderCurrencies() ([]string, error) {
	return mock.currencies, nil
}

func (mock *mockPilvytis) GetPaymentOrderOptions() (*pilvytis.PaymentOrderOptions, error) {
	return &pilvytis.PaymentOrderOptions{
		Minimum: 16.7,
		Suggested: []float64{
			20,
			40,
			100,
		},
	}, nil
}

func (mock *mockPilvytis) GetPaymentGatewayOrder(id identity.Identity, oid string) (*pilvytis.PaymentOrderResponse, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGatewayOrders(id identity.Identity) ([]pilvytis.PaymentOrderResponse, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGatewayOrderInvoice(id identity.Identity, oid string) ([]byte, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGateways() ([]pilvytis.GatewaysResponse, error) {
	return nil, nil
}

type mockPilvytisLocation struct{}

func (mock *mockPilvytisLocation) GetOrigin() locationstate.Location {
	return locationstate.Location{
		Country: "LT",
	}
}

func newMockPilvytisResp(id int, identity, priceC, payC string, recvAmount float64) pilvytis.OrderResponse {
	f := 1.0
	return pilvytis.OrderResponse{
		ID:              uint64(id),
		Status:          "pending",
		Identity:        identity,
		PriceAmount:     &recvAmount,
		MystAmount:      1.0,
		PriceCurrency:   priceC,
		PayAmount:       &f,
		PayCurrency:     &payC,
		PaymentAddress:  "0x00",
		PaymentURL:      "foo.com",
		ReceiveAmount:   &f,
		ReceiveCurrency: "BTC",
		ExpiresAt:       time.Now(),
		CreatedAt:       time.Now(),
	}
}

func TestCreatePaymentOrder(t *testing.T) {
	identity := "0x000000000000000000000000000000000000000b"
	reqBody := contract.OrderRequest{
		MystAmount:  1,
		PayCurrency: "BTC",
	}

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	mockIssuer := &mockPilvytisIssuer{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock, mockIssuer, &mockPilvytisLocation{}).CreatePaymentOrder

	mb, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/identities/%s/payment-order", identity),
		bytes.NewBuffer(mb),
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.POST("/identities/:id/payment-order", handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`{
   "id":1,
   "status":"pending",
   "identity":"0x000000000000000000000000000000000000000b",
   "myst_amount":1,
   "price_amount":1,
   "price_currency":"BTC",
   "pay_amount":1,
   "pay_currency":"BTC",
   "payment_address":"0x00",
   "payment_url":"foo.com",
   "receive_amount":1,
   "receive_currency":"BTC"
}`,
		resp.Body.String(),
	)

}

func TestGetPaymentOrder(t *testing.T) {
	identity := "0x000000000000000000000000000000000000000b"
	orderID := 11

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(orderID, identity, "BTC", "BTC", 1),
	}
	mockIssuer := &mockPilvytisIssuer{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock, mockIssuer, &mockPilvytisLocation{}).GetPaymentOrder

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/identities/%s/payment-order/%d", identity, orderID),
		nil,
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.GET("/identities/:id/payment-order/:order_id", handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		fmt.Sprintf(`{
   "id":%d,
   "status":"pending",
   "identity":"0x000000000000000000000000000000000000000b",
   "myst_amount":1,
   "price_amount":1,
   "price_currency":"BTC",
   "pay_amount":1,
   "pay_currency":"BTC",
   "payment_address":"0x00",
   "payment_url":"foo.com",
   "receive_amount":1,
   "receive_currency":"BTC"
}`, orderID),
		resp.Body.String(),
	)

}

func TestGetPaymentOrders(t *testing.T) {
	identity := "0x000000000000000000000000000000000000000b"

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	mockIssuer := &mockPilvytisIssuer{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock, mockIssuer, &mockPilvytisLocation{}).GetPaymentOrders

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/identities/%s/payment-order", identity),
		nil,
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.POST("/identities/:id/payment-order", handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`[{
   "id":1,
   "status":"pending",
   "identity":"0x000000000000000000000000000000000000000b",
   "myst_amount":1,
   "price_amount":1,
   "price_currency":"BTC",
   "pay_amount":1,
   "pay_currency":"BTC",
   "payment_address":"0x00",
   "payment_url":"foo.com",
   "receive_amount":1,
   "receive_currency":"BTC"
}]`,
		resp.Body.String(),
	)

}

func TestGetCurrency(t *testing.T) {
	mock := &mockPilvytis{currencies: []string{"BTC"}}
	handler := NewPilvytisEndpoint(mock, &mockPilvytisIssuer{}, &mockPilvytisLocation{}).GetPaymentOrderCurrencies

	resp := httptest.NewRecorder()
	url := "/payment-order-currencies"
	req, err := http.NewRequest(
		http.MethodGet,
		url,
		nil,
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.GET(url, handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`["BTC"]`,
		resp.Body.String(),
	)
}

func TestGetPaymentOrderOptions(t *testing.T) {
	mock := &mockPilvytis{}
	handler := NewPilvytisEndpoint(mock, &mockPilvytisIssuer{}, &mockPilvytisLocation{}).GetPaymentOrderOptions

	resp := httptest.NewRecorder()
	url := "/payment-order-options"
	req, err := http.NewRequest(
		http.MethodGet,
		url,
		nil,
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.GET(url, handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`{
			"minimum": 16.7,
			"suggested": [
				20,
				40,
				100
			]
		}`,
		resp.Body.String(),
	)
}
