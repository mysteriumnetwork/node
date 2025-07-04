/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package netstack_provider

import (
	"net"

	"gvisor.dev/gvisor/pkg/tcpip"
)

func parseCIDR(cidrs []string) []*net.IPNet {
	ipnets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		ipnets = append(ipnets, ipnet)
	}
	return ipnets
}

// isPrivateIP returns true if the IP is private / restricted
func (tun *netTun) isPrivateIP(ip net.IP) bool {

	// allow access to local address of Wireguard provider, like 10.182.0.1
	if tun.isLocal(tcpip.AddrFromSlice(ip)) {
		return false
	}

	for _, block := range tun.privateIPv4Blocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
