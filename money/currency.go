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

import "math/big"

// Currency represents a supported currency
type Currency string

// MystSize represents a size of the Myst token.
var MystSize = big.NewInt(1_000_000_000_000_000_000)

const (
	// CurrencyMyst is the myst token currency representation
	CurrencyMyst = Currency("MYST")

	// CurrencyMystt is the test myst token currency representation
	CurrencyMystt = Currency("MYSTT")
)

func (c Currency) String() string {
	return string(c)
}
