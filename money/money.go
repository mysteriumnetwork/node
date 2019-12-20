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

package money

import (
	"fmt"
)

// Money holds the currency type and amount
type Money struct {
	Amount   uint64   `json:"amount,omitempty"`
	Currency Currency `json:"currency,omitempty"`
}

// NewMoney returns a new instance of Money.
// The money is a representation of myst in a uint64 form, with the decimal part expanded.
// This means, that one myst is equivalent to 10 0000 000.
func NewMoney(amount uint64, currency Currency) Money {
	return Money{amount, currency}
}

// String converts struct to string
func (value *Money) String() string {
	return fmt.Sprintf(
		"%d%s",
		value.Amount,
		value.Currency,
	)
}
