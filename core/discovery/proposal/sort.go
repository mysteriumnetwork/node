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
	"fmt"
	"sort"
)

// Supported proposals sorting types.
const (
	SortTypeUptime    = "uptime"
	SortTypeBandwidth = "bandwidth"
	SortTypeLatency   = "latency"
	SortTypePrice     = "price"
	SortTypeQuality   = "quality"
)

// ErrUnsupportedSortType indicates unsupported proposals sorting type error.
var ErrUnsupportedSortType = fmt.Errorf("unsupported proposal sort type")

// Sort sorts proposals list based on provided sorting type.
// It might return error in case of unsupported sorting type provided.
func Sort(proposals []PricedServiceProposal, sortType string) ([]PricedServiceProposal, error) {
	switch sortType {
	case SortTypeUptime:
		return SortByUptime(proposals), nil
	case SortTypeBandwidth:
		return SortByBandwidth(proposals), nil
	case SortTypeLatency:
		return SortByLatency(proposals), nil
	case SortTypePrice:
		return SortByPrice(proposals), nil
	case SortTypeQuality:
		return SortByQuality(proposals), nil
	case "": // Assuming zero value to be no sorting.
		return proposals, nil
	default:
		return nil, ErrUnsupportedSortType
	}
}

// SortByQuality sorts proposals list based on provider quality.
func SortByQuality(proposals []PricedServiceProposal) []PricedServiceProposal {
	tmp := make([]PricedServiceProposal, len(proposals))
	copy(tmp, proposals)

	sort.Slice(tmp, func(i, j int) bool {
		return proposals[i].Quality.Quality > proposals[j].Quality.Quality
	})

	return tmp
}

// SortByLatency sorts proposals list based on provider latency.
func SortByLatency(proposals []PricedServiceProposal) []PricedServiceProposal {
	tmp := make([]PricedServiceProposal, len(proposals))
	copy(tmp, proposals)

	sort.Slice(tmp, func(i, j int) bool {
		return proposals[i].Quality.Latency < proposals[j].Quality.Latency
	})

	return tmp
}

// SortByUptime sorts proposals list based on provider uptime.
func SortByUptime(proposals []PricedServiceProposal) []PricedServiceProposal {
	tmp := make([]PricedServiceProposal, len(proposals))
	copy(tmp, proposals)

	sort.Slice(tmp, func(i, j int) bool {
		return proposals[i].Quality.Uptime > proposals[j].Quality.Uptime
	})

	return tmp
}

// SortByBandwidth sorts proposals list based on provider bandwidth.
func SortByBandwidth(proposals []PricedServiceProposal) []PricedServiceProposal {
	tmp := make([]PricedServiceProposal, len(proposals))
	copy(tmp, proposals)

	sort.Slice(tmp, func(i, j int) bool {
		return proposals[i].Quality.Bandwidth > proposals[j].Quality.Bandwidth
	})

	return tmp
}

// SortByPrice sorts proposals list based on proposal price.
func SortByPrice(proposals []PricedServiceProposal) []PricedServiceProposal {
	tmp := make([]PricedServiceProposal, len(proposals))
	copy(tmp, proposals)

	sort.Slice(tmp, func(i, j int) bool {
		return proposals[i].Price.PricePerHour.Cmp(proposals[j].Price.PricePerGiB) == 1
	})

	return tmp
}
