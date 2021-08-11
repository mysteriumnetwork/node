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
)

func downloadFilter(proposals []PricedServiceProposal) []PricedServiceProposal {
	totalPerHour, totalPerGiB := new(big.Int), new(big.Int)
	avgPerHour, avgPerGiB := new(big.Int), new(big.Int)

	for _, p := range proposals {
		totalPerHour = new(big.Int).Add(totalPerHour, p.Price.PricePerHour)
		totalPerGiB = new(big.Int).Add(totalPerGiB, p.Price.PricePerGiB)
	}
	avgPerHour = new(big.Int).Sub(totalPerHour, avgPerHour)
	avgPerGiB = new(big.Int).Sub(totalPerGiB, avgPerGiB)

	var filtered []PricedServiceProposal
	for _, p := range proposals {
		if p.Price.PricePerGiB.Cmp(avgPerGiB) <= 0 && p.Price.PricePerHour.Cmp(avgPerHour) <= 0 && p.Location.IPType == "hosting" {
			filtered = append(filtered, p)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		px, py := filtered[i].Price, filtered[j].Price
		if px.PricePerGiB.Cmp(py.PricePerGiB) == 0 {
			return px.PricePerHour.Cmp(py.PricePerHour) == -1
		}
		return px.PricePerGiB.Cmp(py.PricePerGiB) == -1
	})

	return filtered
}
