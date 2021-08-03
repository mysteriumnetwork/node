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

package proposal

import (
	"math/big"
	"sort"

	"github.com/mysteriumnetwork/node/market"
)

func browsingFilter(proposals []market.ServiceProposal) []market.ServiceProposal {
	totalPerHour, totalPerGiB := new(big.Int), new(big.Int)
	avgPerHour, avgPerGiB := new(big.Int), new(big.Int)
	var totalQuality, avgQuality float64

	for _, p := range proposals {
		totalPerHour = new(big.Int).Add(totalPerHour, p.Price.PerHour)
		totalPerGiB = new(big.Int).Add(totalPerGiB, p.Price.PerGiB)
		totalQuality += p.Quality.Quality
	}
	avgPerHour = new(big.Int).Sub(totalPerHour, avgPerHour)
	avgPerGiB = new(big.Int).Sub(totalPerGiB, avgPerGiB)
	avgQuality = avgQualityCeiling(totalQuality / float64(len(proposals)))

	var filtered []market.ServiceProposal
	for _, p := range proposals {
		if p.Price.PerGiB.Cmp(avgPerGiB) <= 0 && p.Price.PerHour.Cmp(avgPerHour) <= 0 && p.Quality.Quality > avgQuality {
			filtered = append(filtered, p)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		qx, qy := filtered[i].Quality, filtered[j].Quality
		px, py := filtered[i].Price, filtered[j].Price
		if qx.Quality == qy.Quality {
			if px.PerGiB.Cmp(py.PerGiB) == 0 {
				return px.PerHour.Cmp(py.PerHour) == -1
			}
			return px.PerGiB.Cmp(py.PerGiB) == -1
		}
		return qx.Quality > qy.Quality
	})

	return filtered
}
