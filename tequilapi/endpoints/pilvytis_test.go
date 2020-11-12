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

	"github.com/julienschmidt/httprouter"
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
}

func (mock *mockPilvytis) CreatePaymentOrder(i identity.Identity, mystAmount float64, payCurrency string, ln bool) (pilvytis.OrderResponse, error) {
	if i.Address != mock.identity {
		return pilvytis.OrderResponse{}, errors.New("wrong identity")
	}

	return mock.resp, nil
}

func (mock *mockPilvytis) GetPaymentOrder(i identity.Identity, oid uint64) (pilvytis.OrderResponse, error) {
	if i.Address != mock.identity {
		return pilvytis.OrderResponse{}, errors.New("wrong identity")
	}

	return mock.resp, nil
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
	params := httprouter.Params{{Key: "id", Value: identity}}
	reqBody := contract.OrderRequest{
		MystAmount:  1,
		PayCurrency: "BTC",
	}

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock).CreatePaymentOrder

	mb, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/test",
		bytes.NewBuffer(mb),
	)
	assert.NoError(t, err)

	handler(resp, req, params)
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
	id := 11
	params := httprouter.Params{
		{Key: "id", Value: identity},
		{Key: "order_id", Value: fmt.Sprint(id)},
	}

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(id, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock).GetPaymentOrder

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)
	assert.NoError(t, err)

	handler(resp, req, params)
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
}`, id),
		resp.Body.String(),
	)

}

func TestGetPaymentOrders(t *testing.T) {
	identity := "0x000000000000000000000000000000000000000b"
	params := httprouter.Params{
		{Key: "id", Value: identity},
	}

	mock := &mockPilvytis{
		identity: identity,
		resp:     newMockPilvytisResp(1, identity, "BTC", "BTC", 1),
	}
	handler := NewPilvytisEndpoint(mock).GetPaymentOrders

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)
	assert.NoError(t, err)

	handler(resp, req, params)
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
	handler := NewPilvytisEndpoint(mock).GetPaymentOrderCurrencies

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)
	assert.NoError(t, err)

	handler(resp, req, httprouter.Params{})
	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`["BTC"]`,
		resp.Body.String(),
	)

}
