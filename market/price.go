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
	"time"
)

// LatestPrices represents the latest pricing.
type LatestPrices struct {
	Defaults           *PriceHistory            `json:"defaults"`
	PerCountry         map[string]*PriceHistory `json:"per_country"`
	CurrentValidUntil  time.Time                `json:"current_valid_until"`
	PreviousValidUntil time.Time                `json:"previous_valid_until"`
	CurrentServerTime  time.Time                `json:"current_server_time,omitempty"`
}

// PriceHistory represents the current and previous price.
type PriceHistory struct {
	Current  *PriceByType `json:"current"`
	Previous *PriceByType `json:"previous"`
}

// PriceByType is a slice of pricing by type.
type PriceByType struct {
	Residential *PriceByServiceType `json:"residential"`
	Other       *PriceByServiceType `json:"other"`
}

// PriceByServiceType is a slice of pricing by service type.
type PriceByServiceType struct {
	Wireguard    *Price `json:"wireguard"`
	Scraping     *Price `json:"scraping"`
	DataTransfer *Price `json:"data_transfer"`
	DVPN         *Price `json:"dvpn"`
}

// Price represents the price.
type Price struct {
	PricePerHour *big.Int `json:"price_per_hour"`
	PricePerGiB  *big.Int `json:"price_per_gib"`
}

// IsFree Determines if the price has any values set or not.
func (p Price) IsFree() bool {
	return p.PricePerGiB.Cmp(big.NewInt(0)) == 0 && p.PricePerHour.Cmp(big.NewInt(0)) == 0
}

// NewPrice creates a new Price instance.
func NewPrice(perHour, perGiB int64) *Price {
	return &Price{
		PricePerHour: big.NewInt(perHour),
		PricePerGiB:  big.NewInt(perGiB),
	}
}

func (p Price) String() string {
	return p.PricePerHour.String() + "/h, " + p.PricePerGiB.String() + "/GiB "
}
