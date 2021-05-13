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
func NewPrice(perHour, perGiB int64, currency money.Currency) *Price {
	return &Price{
		Currency: currency,
		PerHour:  big.NewInt(perHour),
		PerGiB:   big.NewInt(perGiB),
	}
}

// NewPriceB creates a new Price instance using big.Ints.
func NewPriceB(perHour, perGiB *big.Int, currency money.Currency) *Price {
	p := NewPrice(0, 0, currency)
	if perHour != nil {
		p.PerHour = perHour
	}
	if perGiB != nil {
		p.PerGiB = perGiB
	}
	return p
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
