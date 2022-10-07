/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package contract

// IPDTO describes IP metadata.
// swagger:model IPDTO
type IPDTO struct {
	// public IP address
	// example: 127.0.0.1
	IP string `json:"ip"`
}

// LocationDTO describes IP location metadata.
// swagger:model LocationDTO
type LocationDTO struct {
	// IP address
	// example: 1.2.3.4
	IP string `json:"ip"`
	// Autonomous system number
	// example: 62179
	ASN int `json:"asn"`
	// Internet Service Provider name
	// example: Telia Lietuva, AB
	ISP string `json:"isp"`

	// Continent
	// example: EU
	Continent string `json:"continent"`
	// Node Country
	// example: LT
	Country string `json:"country"`
	// Node Region
	// example: Vilnius region
	Region string `json:"region"`
	// Node City
	// example: Vilnius
	City string `json:"city"`

	// IP type (data_center, residential, etc.)
	// example: residential
	IPType string `json:"ip_type"`
}
