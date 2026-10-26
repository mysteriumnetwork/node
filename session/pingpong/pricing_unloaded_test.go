/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/market"
)

// Example values only; not taken from any real node or price list.
const testCountryUnloaded = "XX"

type unreachableDiscoPricing struct{}

func (unreachableDiscoPricing) GetPricing() (market.LatestPrices, error) {
	return market.LatestPrices{}, errors.New("discovery unreachable")
}

// A pricer that never managed to load prices from discovery only has the
// built-in defaults, which carry no previous price. A consumer price that is not
// valid must be refused, not crash the price check.
func TestPricer_NeverLoaded_NonDefaultPriceIsRefusedWithoutPanic(t *testing.T) {
	p := NewPricer(unreachableDiscoPricing{})

	// Example price above the built-in default, so it is not valid with or
	// without loaded prices.
	in := market.Price{
		PricePerHour: new(big.Int).Mul(defaultPrice.PricePerHour, big.NewInt(10)),
		PricePerGiB:  new(big.Int).Mul(defaultPrice.PricePerGiB, big.NewInt(10)),
	}

	for _, nodeType := range []string{"residential", "hosting"} {
		var valid bool
		assert.NotPanics(t, func() {
			valid = p.IsPriceValid(in, nodeType, testCountryUnloaded, "wireguard")
		}, nodeType)
		assert.False(t, valid, nodeType)
	}
}

// The built-in default price itself is still accepted while discovery is unreachable.
func TestPricer_NeverLoaded_DefaultPriceIsAccepted(t *testing.T) {
	p := NewPricer(unreachableDiscoPricing{})

	in := market.Price{PricePerHour: defaultPrice.PricePerHour, PricePerGiB: defaultPrice.PricePerGiB}

	var valid bool
	assert.NotPanics(t, func() {
		valid = p.IsPriceValid(in, "residential", testCountryUnloaded, "wireguard")
	})
	assert.True(t, valid)
}
