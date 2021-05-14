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

package market

import (
	"math/big"
	"time"
)

// LatestPrices contains the current and previous prices.
type LatestPrices struct {
	Current  *Prices `json:"current"`
	Previous *Prices `json:"previous"`
}

// Prices represents the per hour and per byte prices.
type Prices struct {
	ValidUntil   time.Time `json:"valid_until"`
	PricePerHour *big.Int  `json:"price_per_hour"`
	PricePerGiB  *big.Int  `json:"price_per_gib"`
}

// IsFree Determines if the price has any values set or not.
func (p Prices) IsFree() bool {
	return p.PricePerGiB.Cmp(big.NewInt(0)) == 0 && p.PricePerHour.Cmp(big.NewInt(0)) == 0
}

// NewPrice creates a new Price instance.
func NewPrice(perHour, perGiB int64) *Prices {
	return &Prices{
		PricePerHour: big.NewInt(perHour),
		PricePerGiB:  big.NewInt(perGiB),
	}
}
