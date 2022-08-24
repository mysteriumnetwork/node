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

package pilvytis

// OrderIssuer combines the pilvytis API and order tracker.
// Only the order issuer can issue new payment orders.
type OrderIssuer struct {
	api     *API
	tracker *StatusTracker
}

// NewOrderIssuer returns a new order issuer.
func NewOrderIssuer(api *API, tracker *StatusTracker) *OrderIssuer {
	return &OrderIssuer{
		api:     api,
		tracker: tracker,
	}
}

// CreatePaymentGatewayOrder will create a new payment order and send a notification to start tracking it.
func (o *OrderIssuer) CreatePaymentGatewayOrder(cgo GatewayOrderRequest) (*GatewayOrderResponse, error) {
	resp, err := o.api.createPaymentGatewayOrder(cgo)
	if err != nil {
		return nil, err
	}
	o.tracker.UpdateOrdersFor(cgo.Identity)

	return resp, err
}
