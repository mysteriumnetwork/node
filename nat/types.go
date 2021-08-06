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

package nat

// NATType represents nat type
type NATType string

// Enum of DiscoverNATBehavior return values
const (
	NATTypeNone               NATType = "none"
	NATTypeFullCone           NATType = "fullcone"
	NATTypeRestrictedCone     NATType = "rcone"
	NATTypePortRestrictedCone NATType = "prcone"
	NATTypeSymmetric          NATType = "symmetric"
)

// HumanReadableTypes is a map of enum values to human readable strings
var HumanReadableTypes = map[NATType]string{
	NATTypeNone:               "None",
	NATTypeFullCone:           "Full Cone",
	NATTypeRestrictedCone:     "Restricted Cone",
	NATTypePortRestrictedCone: "Port Restricted Cone",
	NATTypeSymmetric:          "Symmetric",
}
