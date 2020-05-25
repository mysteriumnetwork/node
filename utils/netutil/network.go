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

package netutil

import (
	"net"
)

// AssignIP assigns subnet to given interface.
func AssignIP(iface string, subnet net.IPNet) error {
	return assignIP(iface, subnet)
}

// ExcludeRoute excludes given IP from VPN tunnel.
func ExcludeRoute(ip net.IP) error {
	return excludeRoute(ip)
}

// AddDefaultRoute adds default VPN tunnel route.
func AddDefaultRoute(iface string) error {
	return addDefaultRoute(iface)
}
