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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/payments/exchange"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockPilvytis struct {
	currencies []string
	identity   string
	respgw     pilvytis.GatewayOrderResponse
}

type mockPilvytisIssuer struct {
	identity string
	respgw   pilvytis.GatewayOrderResponse
}

func (mock *mockPilvytisIssuer) CreatePaymentGatewayOrder(cgo pilvytis.GatewayOrderRequest) (*pilvytis.GatewayOrderResponse, error) {
	if cgo.Identity.Address != mock.identity {
		return nil, errors.New("wrong identity")
	}

	return &mock.respgw, nil
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

func (mock *mockPilvytis) GetPaymentGatewayOrder(id identity.Identity, oid string) (*pilvytis.GatewayOrderResponse, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGatewayOrders(id identity.Identity) ([]pilvytis.GatewayOrderResponse, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGatewayOrderInvoice(id identity.Identity, oid string) ([]byte, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetPaymentGateways(_ exchange.Currency) ([]pilvytis.GatewaysResponse, error) {
	return nil, nil
}

func (mock *mockPilvytis) GetRegistrationPaymentStatus(id identity.Identity) (*pilvytis.RegistrationPaymentResponse, error) {
	if id.Address != mock.identity {
		return &pilvytis.RegistrationPaymentResponse{
			Paid: false,
		}, nil
	}

	return &pilvytis.RegistrationPaymentResponse{
		Paid: true,
	}, nil
}

func (mock *mockPilvytis) GatewayClientCallback(id identity.Identity, gateway string, payload any) error {
	return nil
}

type mockPilvytisLocation struct{}

func (mock *mockPilvytisLocation) GetOrigin() locationstate.Location {
	return locationstate.Location{
		Country: "LT",
	}
}

func TestGetRegistrationPaymentStatus(t *testing.T) {
	identity := "0x000000000000000000000000000000000000000b"

	mock := &mockPilvytis{
		identity: identity,
	}
	handler := NewPilvytisEndpoint(mock, &mockPilvytisIssuer{}, &mockPilvytisLocation{}).GetRegistrationPaymentStatus

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/identities/%s/registration-payment", identity),
		nil,
	)
	assert.NoError(t, err)

	g := gin.Default()
	g.GET("/identities/:id/registration-payment", handler)
	g.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.JSONEq(t,
		`{
			"paid":true
		 }`,
		resp.Body.String(),
	)
}
