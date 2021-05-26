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

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mysteriumnetwork/node/money"
)

// Price represents the proposal price.
type Price struct {
	Currency money.Currency `json:"currency"`
	PerHour  *big.Int       `json:"per_hour"`
	PerGiB   *big.Int       `json:"per_gib"`
}

// NewPrice creates a new Price instance.
func NewPrice(perHour, perGiB uint64, currency money.Currency) Price {
	return Price{
		Currency: currency,
		PerHour:  new(big.Int).SetUint64(perHour),
		PerGiB:   new(big.Int).SetUint64(perGiB),
	}
}

// NewPricePtr returns a pointer to a new Price instance.
func NewPricePtr(perHour, perGiB uint64, currency money.Currency) *Price {
	price := NewPrice(perHour, perGiB, currency)
	return &price
}

// IsFree returns true if the service pricing is set to 0.
func (p Price) IsFree() bool {
	return p.PerHour.Cmp(big.NewInt(0)) == 0 && p.PerGiB.Cmp(big.NewInt(0)) == 0
}

// Validate validates the price.
func (p Price) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Currency, validation.Required),
		validation.Field(&p.PerHour, validation.Required),
		validation.Field(&p.PerGiB, validation.Required),
	)
}

func (p Price) String() string {
	return p.PerHour.String() + "/h, " + p.PerGiB.String() + "/GiB, " + "Currency: " + string(p.Currency)
}
