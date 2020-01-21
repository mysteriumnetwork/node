/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package noop

import (
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

// ServiceType indicates "noop" service type
const ServiceType = "noop"

// ServiceDefinition structure represents "noop" service parameters
type ServiceDefinition struct {
	// Approximate information on location where the service is provided from
	Location market.Location `json:"location"`
}

// GetLocation returns geographic location of service definition provider
func (service ServiceDefinition) GetLocation() market.Location {
	return service.Location
}

// PaymentMethodNoop indicates payment method without payment at all
const PaymentMethodNoop = "NOOP"

// PaymentNoop structure describes 0 price for Noop payment
type PaymentNoop struct {
	Price money.Money `json:"price"`
}

// GetPrice returns price of payment per time
func (method PaymentNoop) GetPrice() money.Money {
	return method.Price
}

// GetType returns NOOP
func (method PaymentNoop) GetType() string {
	return PaymentMethodNoop
}

// GetRate returns the payment rate
func (method PaymentNoop) GetRate() market.PaymentRate {
	return market.PaymentRate{
		PerTime: time.Minute,
		PerByte: 1000000,
	}
}
