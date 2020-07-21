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

package mocks

import (
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

// PaymentMethod is a test-friendly payment method.
type PaymentMethod struct {
	Rate        market.PaymentRate
	PaymentType string
	Price       money.Money
}

// GetPrice returns pre-defined price.
func (mpm *PaymentMethod) GetPrice() money.Money {
	return mpm.Price
}

// GetType returns pre-defined type.
func (mpm *PaymentMethod) GetType() string {
	return mpm.PaymentType
}

// GetRate returns pre-defined rate.
func (mpm *PaymentMethod) GetRate() market.PaymentRate {
	return mpm.Rate
}

// DefaultPaymentMethod is a mock default payment method (workaround package import cycles).
func DefaultPaymentMethod() *PaymentMethod {
	return &PaymentMethod{
		Rate: market.PaymentRate{
			PerTime: time.Minute,
			PerByte: 7669584,
		},
		PaymentType: "BYTES_TRANSFERRED_WITH_TIME",
		Price:       money.Money{Amount: big.NewInt(50000), Currency: money.CurrencyMyst},
	}
}

// DefaultPaymentMethodType is a mock default.
const DefaultPaymentMethodType = "BYTES_TRANSFERRED_WITH_TIME"
