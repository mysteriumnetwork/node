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

package locationstate

// Location structure represents location information
type Location struct {
	IP  string `json:"ip"`
	ASN int    `json:"asn"`
	ISP string `json:"isp"`

	Continent string `json:"continent"`
	Country   string `json:"country"`
	Region    string `json:"region"`
	City      string `json:"city"`

	IPType string `json:"ip_type"`
}
