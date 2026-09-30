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
	"math"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const (
	video720pMBPerMin   = 15
	audioNormalMBPerMin = 0.75
	browsingMBPerMin    = 0.5
)

// Estimates represent estimated entertainment
type Estimates struct {
	VideoMinutes    uint64
	MusicMinutes    uint64
	BrowsingMinutes uint64
	// TrafficMB is decimal megabytes (1 MB = 1,000,000 bytes), not MiB.
	TrafficMB   uint64
	PricePerGiB float64
	PricePerMin float64
}

// PriceProvider supplies current consumer prices from the node's shared pricing cache.
type PriceProvider interface {
	GetCurrentPrice(nodeType, country, serviceType string) (market.Price, error)
}

// Estimator estimates usage from consumer prices.
type Estimator struct {
	prices      PriceProvider
	pricePerGiB float64
	pricePerMin float64
}

// NewMarketEstimator uses current residential WireGuard pricing on each estimate.
// No destination is selected by these APIs, so use the market's global default
// rather than a country-specific rate or the consumer's own location.
func NewMarketEstimator(prices PriceProvider) *Estimator {
	return &Estimator{prices: prices}
}

// NewEstimator constructs an estimator with explicit MYST/GiB and MYST/minute rates.
func NewEstimator(pricePerGiB, pricePerMin float64) *Estimator {
	return &Estimator{
		pricePerGiB: pricePerGiB,
		pricePerMin: pricePerMin,
	}
}

// EstimatedEntertainment calculates average service times
func (e *Estimator) EstimatedEntertainment(myst float64) Estimates {
	if e.prices != nil {
		price, err := e.prices.GetCurrentPrice("residential", "", "wireguard")
		if err != nil {
			log.Warn().Err(err).Msg("could not obtain entertainment pricing")
			return Estimates{}
		}
		if price.PricePerGiB == nil || price.PricePerHour == nil {
			return Estimates{}
		}
		// Keep rates local: estimates may run concurrently with pricing updates.
		return NewEstimator(crypto.BigMystToFloat(price.PricePerGiB),
			crypto.BigMystToFloat(price.PricePerHour)/60).EstimatedEntertainment(myst)
	}
	if myst < 0 || math.IsNaN(myst) || math.IsInf(myst, 0) ||
		e.pricePerGiB <= 0 || math.IsNaN(e.pricePerGiB) || math.IsInf(e.pricePerGiB, 0) ||
		e.pricePerMin < 0 || math.IsNaN(e.pricePerMin) || math.IsInf(e.pricePerMin, 0) {
		return Estimates{}
	}
	return Estimates{
		VideoMinutes:    e.minutes(myst, video720pMBPerMin),
		MusicMinutes:    e.minutes(myst, audioNormalMBPerMin),
		BrowsingMinutes: e.minutes(myst, browsingMBPerMin),
		TrafficMB:       uint64(mib2MB(e.totalTrafficMiB(myst))),
		PricePerGiB:     e.pricePerGiB,
		PricePerMin:     e.pricePerMin,
	}
}

func mib2MB(mibs float64) float64 {
	return mibs * math.Pow(2, 20) / math.Pow(10, 6)
}

func mb2MiB(mb float64) float64 {
	return mb * math.Pow(10, 6) / math.Pow(2, 20)
}

func (e *Estimator) totalTrafficMiB(amount float64) float64 {
	return amount / e.pricePerGiB * 1024
}

func (e *Estimator) minutes(amount, serviceMBPerMin float64) uint64 {
	pricePerMiB := e.pricePerGiB / 1024
	totalPricePerMin := mb2MiB(serviceMBPerMin)*pricePerMiB + e.pricePerMin
	return uint64(amount / totalPricePerMin)
}
