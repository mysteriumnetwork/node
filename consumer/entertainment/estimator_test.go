/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package entertainment

import (
	"errors"
	"math"
	"testing"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/payments/crypto"

	"github.com/stretchr/testify/assert"
)

func TestEstimator(t *testing.T) {
	// given
	estimator := NewEstimator(0.01, 0.0001)

	// expect
	assert.Equal(t, 102400.0, estimator.totalTrafficMiB(1))
	assert.Equal(t, 204800.0, estimator.totalTrafficMiB(2))

	assert.Equal(t, uint64(106), estimator.minutes(1, 1000))
	assert.Equal(t, uint64(53), estimator.minutes(1, 2000))
	assert.Equal(t, uint64(9555), estimator.minutes(1, 0.5))

	// and
	e := estimator.EstimatedEntertainment(5)
	assert.Equal(t, uint64(536870), e.TrafficMB)
	assert.Equal(t, 0.01, e.PricePerGiB)
	assert.Equal(t, 0.0001, e.PricePerMin)

	// can fluctuate based on constants so just assert it's set
	assert.Less(t, uint64(0), e.VideoMinutes)
	assert.Less(t, uint64(0), e.MusicMinutes)
	assert.Less(t, uint64(0), e.BrowsingMinutes)
}

// The provider is queried for every estimate, so a market refresh is reflected
// without restarting either the mobile node or TequilAPI.
type testPriceProvider struct {
	t     *testing.T
	price market.Price
	err   error
}

func (p *testPriceProvider) GetCurrentPrice(nodeType, country, serviceType string) (market.Price, error) {
	assert.Equal(p.t, "residential", nodeType)
	assert.Empty(p.t, country)
	assert.Equal(p.t, "wireguard", serviceType)
	return p.price, p.err
}

func TestMarketEstimator(t *testing.T) {
	provider := &testPriceProvider{t: t, price: market.Price{
		PricePerGiB:  crypto.FloatToBigMyst(3.73),
		PricePerHour: crypto.FloatToBigMyst(0.06),
	}}
	estimator := NewMarketEstimator(provider)
	estimate := estimator.EstimatedEntertainment(29.444)
	assert.InDelta(t, 3.73, estimate.PricePerGiB, 1e-12)
	assert.InDelta(t, 0.001, estimate.PricePerMin, 1e-12)
	// Convert decimal MB back to GiB, allowing truncation of less than 1 MB.
	assert.InDelta(t, 29.444/3.73, float64(estimate.TrafficMB)*1e6/(1<<30), 0.001)
	assert.Equal(t, uint64(math.Floor(29.444/(15e6/(1<<30)*3.73+0.001))), estimate.VideoMinutes)

	provider.price.PricePerGiB = crypto.FloatToBigMyst(7.46)
	refreshed := estimator.EstimatedEntertainment(29.444)
	assert.InDelta(t, float64(estimate.TrafficMB)/2, float64(refreshed.TrafficMB), 1)
	assert.InDelta(t, 7.46, refreshed.PricePerGiB, 1e-12)
}

func TestEstimatorUnits(t *testing.T) {
	assert.Equal(t, 1048.576, mib2MB(1000))
	assert.Equal(t, 1000.0, mb2MiB(1048.576))
	estimate := NewEstimator(1, 0).EstimatedEntertainment(1)
	assert.Equal(t, uint64(1073), estimate.TrafficMB)  // 1 GiB = 1073.741824 MB
	assert.Equal(t, uint64(71), estimate.VideoMinutes) // 15 decimal MB/minute
}

func TestMarketEstimatorUnavailablePrice(t *testing.T) {
	provider := &testPriceProvider{t: t, err: errors.New("unavailable")}
	assert.Equal(t, Estimates{}, NewMarketEstimator(provider).EstimatedEntertainment(29.444))
	provider.err = nil
	assert.Equal(t, Estimates{}, NewMarketEstimator(provider).EstimatedEntertainment(29.444))
}

func TestEstimatorInvalidAmount(t *testing.T) {
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1)} {
		assert.Equal(t, Estimates{}, NewEstimator(3.73, 0).EstimatedEntertainment(amount))
	}
}
