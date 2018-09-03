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
	"github.com/mysterium/node/datasize"
	"github.com/mysterium/node/money"
)

// PaymentMethodPerBytes indicates payment method for data amount transferred
const PaymentMethodPerBytes = "PER_BYTES"

// PaymentPerBytes structure describes price per unit and how much bytes were transferred
type PaymentPerBytes struct {
	Price money.Money `json:"price"`

	// Service bytes provided for paid price
	Bytes datasize.BitSize `json:"bytes,omitempty"`
}

// GetPrice returns payment price
func (method PaymentPerBytes) GetPrice() money.Money {
	return method.Price
}
