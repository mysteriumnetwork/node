/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package location

// ServiceLocationInfo represents data needed to determine location and if service is behind the NAT
type ServiceLocationInfo struct {
	OutIP   string
	PubIP   string
	Country string
}

// BehindNAT checks if service is behind NAT network.
func (loc ServiceLocationInfo) BehindNAT() bool {
	return loc.OutIP != loc.PubIP
}
