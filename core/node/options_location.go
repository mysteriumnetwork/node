/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package node

// LocationType identifies location
type LocationType string

const (
	// LocationTypeManual defines type which resolves location from manually entered values
	LocationTypeManual = LocationType("manual")
	// LocationTypeBuiltin defines type which resolves location from built in DB
	LocationTypeBuiltin = LocationType("builtin")
	// LocationTypeMMDB defines type which resolves location from given MMDB file
	LocationTypeMMDB = LocationType("mmdb")
	// LocationTypeOracle defines type which resolves location from given URL of LocationOracle
	LocationTypeOracle = LocationType("oracle")
)

// OptionsLocation describes possible parameters of location detection configuration
type OptionsLocation struct {
	IPDetectorURL string

	Type     LocationType
	Address  string
	Country  string
	City     string
	NodeType string
}
