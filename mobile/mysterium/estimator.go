/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.

 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package mysterium

import "github.com/mysteriumnetwork/node/consumer/entertainment"

// Estimates represent estimated entertainment
type Estimates struct {
	VideoMinutes    int64
	MusicMinutes    int64
	BrowsingMinutes int64
	TrafficMB       int64
	PricePerGB      float64
	PricePerMin     float64
}

func newEstimates(e entertainment.Estimates) *Estimates {
	return &Estimates{
		VideoMinutes:    int64(e.VideoMinutes),
		MusicMinutes:    int64(e.MusicMinutes),
		BrowsingMinutes: int64(e.BrowsingMinutes),
		TrafficMB:       int64(e.TrafficMB),
		PricePerGB:      e.PricePerGiB,
		PricePerMin:     e.PricePerMin,
	}
}

// CalculateEstimates calculates average service times
func (mb *MobileNode) CalculateEstimates(amount float64) *Estimates {
	return newEstimates(mb.entertainmentEstimator.EstimatedEntertainment(amount))
}
