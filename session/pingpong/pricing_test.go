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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/market"
)

type mockDiscoPricing struct {
	prices market.LatestPrices
	err    error
}

func (m *mockDiscoPricing) GetPricing() (market.LatestPrices, error) {
	return m.prices, m.err
}

func pricesForAllServices(p *market.Price) *market.PriceByServiceType {
	return &market.PriceByServiceType{
		Wireguard:    p,
		Scraping:     p,
		DataTransfer: p,
		DVPN:         p,
		Monitoring:   p,
	}
}

// Test values are examples only.
const testCountry = "XX"

func newTestPricer(current, previous *market.Price) *Pricer {
	now := time.Now().UTC()
	return NewPricer(&mockDiscoPricing{
		prices: market.LatestPrices{
			Defaults: &market.PriceHistory{
				Current: &market.PriceByType{
					Residential: pricesForAllServices(current),
					Other:       pricesForAllServices(current),
				},
				Previous: &market.PriceByType{
					Residential: pricesForAllServices(previous),
					Other:       pricesForAllServices(previous),
				},
			},
			CurrentValidUntil:  now.Add(time.Hour),
			PreviousValidUntil: now,
			CurrentServerTime:  now,
		},
	})
}

func TestPricer_IsPriceValid(t *testing.T) {
	current := market.NewPrice(500_000_000_000_000, 5_000_000_000_000_000_000)
	previous := market.NewPrice(400_000_000_000_000, 4_000_000_000_000_000_000)
	free := market.NewPrice(0, 0)
	oneWei := market.NewPrice(1, 1)
	underpriced := market.NewPrice(60_000_000_000_000, 10_000_000_000_000_000)
	builtInDefault := market.Price{PricePerHour: defaultPrice.PricePerHour, PricePerGiB: defaultPrice.PricePerGiB}
	above := market.NewPrice(500_000_000_000_000, 6_000_000_000_000_000_000)

	tests := []struct {
		name        string
		listed      *market.Price
		in          market.Price
		nodeType    string
		serviceType string
		want        bool
	}{
		{name: "current price accepted", listed: current, in: *current, nodeType: "residential", serviceType: "wireguard", want: true},
		{name: "previous price accepted", listed: current, in: *previous, nodeType: "residential", serviceType: "wireguard", want: true},
		{name: "current price accepted, hosting", listed: current, in: *current, nodeType: "hosting", serviceType: "wireguard", want: true},
		{name: "free accepted when discovery lists free", listed: free, in: *free, nodeType: "residential", serviceType: "wireguard", want: true},
		{name: "free refused", listed: current, in: *free, nodeType: "residential", serviceType: "wireguard", want: false},
		{name: "free refused, hosting", listed: current, in: *free, nodeType: "hosting", serviceType: "wireguard", want: false},
		{name: "free refused, noop maps to data transfer price", listed: current, in: *free, nodeType: "residential", serviceType: "noop", want: false},
		{name: "one wei refused", listed: current, in: *oneWei, nodeType: "residential", serviceType: "wireguard", want: false},
		{name: "underpriced refused", listed: current, in: *underpriced, nodeType: "residential", serviceType: "wireguard", want: false},
		{name: "built-in default refused when discovery lists another price", listed: current, in: builtInDefault, nodeType: "residential", serviceType: "wireguard", want: false},
		{name: "price above current refused", listed: current, in: *above, nodeType: "residential", serviceType: "wireguard", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricer := newTestPricer(tt.listed, previous)
			got := pricer.IsPriceValid(tt.in, tt.nodeType, testCountry, tt.serviceType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPricer_IsPriceValid_PricesNeverLoaded(t *testing.T) {
	pricer := NewPricer(&mockDiscoPricing{err: errors.New("discovery unreachable")})
	builtInDefault := market.Price{PricePerHour: defaultPrice.PricePerHour, PricePerGiB: defaultPrice.PricePerGiB}

	assert.True(t, pricer.IsPriceValid(builtInDefault, "residential", testCountry, "wireguard"),
		"a node that never loaded prices still accepts the built-in default price")
}

func TestPricer_IsPriceValid_PaymentsDebugAcceptsAnything(t *testing.T) {
	config.Current.SetUser(config.FlagPaymentsDuringSessionDebug.Name, true)
	defer config.Current.RemoveUser(config.FlagPaymentsDuringSessionDebug.Name)

	pricer := newTestPricer(market.NewPrice(500_000_000_000_000, 5_000_000_000_000_000_000), market.NewPrice(1, 1))
	assert.True(t, pricer.IsPriceValid(*market.NewPrice(0, 0), "residential", testCountry, "wireguard"))
}
