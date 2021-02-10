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
	"math/big"

	"github.com/mysteriumnetwork/node/config"
)

// Money holds the currency type and amount
type Money struct {
	Amount   *big.Int `json:"amount,omitempty"`
	Currency Currency `json:"currency,omitempty"`
}

// New returns a new instance of Money.
// Expected `amount` value for 1 myst is equal to 1_000_000_000_000_000_000.
// It also allows for an optional currency value to be passed,
// if one is not passed, default config value is used.
func New(amount *big.Int, currency ...Currency) Money {
	m := Money{
		Amount: amount,
	}

	if len(currency) > 0 {
		m.Currency = currency[0]
	} else {
		m.Currency = Currency(config.GetString(config.FlagDefaultCurrency))
	}

	return m
}

// String converts Money struct into a string
// which is represented by a float64 with 6 number precision.
func (value Money) String() string {
	amount := new(big.Float).SetInt(value.Amount)
	size := new(big.Float).SetInt(MystSize)
	val, _ := new(big.Float).Quo(amount, size).Float64()
	return fmt.Sprintf(
		"%.6f%s",
		val,
		value.Currency,
	)
}
