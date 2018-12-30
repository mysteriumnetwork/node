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

type Money struct {
	Amount   uint64   `json:"amount,omitempty"`
	Currency Currency `json:"currency,omitempty"`
}

func NewMoney(amount float64, currency Currency) Money {
	return Money{uint64(amount * 100000000), currency}
}

// String converts struct to string
func (value *Money) String() string {
	return fmt.Sprintf(
		"%d%s",
		value.Amount,
		value.Currency,
	)
}
