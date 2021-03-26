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

import "math"

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
	TrafficMB       uint64
	PricePerGiB     float64
	PricePerMin     float64
}

// Estimator stores average provider prices to estimate entertainment estimates
type Estimator struct {
	pricePerGiB float64
	pricePerMin float64
}

// NewEstimator constructor
func NewEstimator(pricePerGiB, pricePerMin float64) *Estimator {
	return &Estimator{
		pricePerGiB: pricePerGiB,
		pricePerMin: pricePerMin,
	}
}

// EstimatedEntertainment calculates average service times
func (e *Estimator) EstimatedEntertainment(myst float64) Estimates {
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
