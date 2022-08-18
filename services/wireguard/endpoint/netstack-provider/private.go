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
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"gvisor.dev/gvisor/pkg/tcpip"
)

func mustParseCIDR(cidrs []string) []*net.IPNet {
	ipnets := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		ipnets[i] = ipnet
	}
	return ipnets
}

var privateIPv4Block []*net.IPNet

func initPrivateIPList() {
	privateIPv4Block = mustParseCIDR(strings.Split(config.FlagFirewallProtectedNetworks.GetValue(), ","))
}

// isPublicAddr retruns true if the IP is private / restricted
func (tun *netTun) isPrivateIP(ip net.IP) bool {

	// allow access to local address of Wireguard provider, like 10.182.0.1
	if tun.isLocal(tcpip.Address(ip)) {
		return false
	}

	for _, block := range privateIPv4Block {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
