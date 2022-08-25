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

package dns

import (
	"fmt"

	"github.com/miekg/dns"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// configuration returns the system DNS configuration.
func configuration() (*dns.ClientConfig, error) {
	addrs, err := winipcfg.GetAdaptersAddresses(2, 0x0010)
	if err != nil {
		return nil, fmt.Errorf("error getting adapters addresses: %w", err)
	}

	resolvers := map[string]bool{}
	for _, addr := range addrs {
		for dnsServer := addr.FirstDNSServerAddress; dnsServer != nil; dnsServer = dnsServer.Next {
			ip := dnsServer.Address.IP()
			resolvers[ip.String()] = true
		}
	}

	servers := []string{}
	for server := range resolvers {
		servers = append(servers, server)
	}

	return &dns.ClientConfig{
		Servers:  servers,
		Port:     "53",
		Ndots:    1,
		Timeout:  5,
		Attempts: 2,
	}, nil
}

// ConfiguredServers returns DNS server IPs from the system DNS configuration.
func ConfiguredServers() ([]string, error) {
	config, err := configuration()
	if err != nil {
		return nil, err
	}
	return config.Servers, nil
}
