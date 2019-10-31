/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dto

import (
	"time"

	"github.com/mysteriumnetwork/node/money"
)

// PaymentMethodPerTime defines payment method for used amount of time of service
const PaymentMethodPerTime = "PER_TIME"

// PaymentRate structure defines price and amount of time used of service
type PaymentRate struct {
	Price money.Money `json:"price"`

	// Service duration provided for paid price
	Duration time.Duration `json:"duration"`
}

// GetPrice returns price of payment per time
func (method PaymentRate) GetPrice() money.Money {
	return method.Price
}
