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

package requests

import (
	"context"
	"net"
)

// ResolveContext specifies the resolve function for doing custom DNS lookup.
type ResolveContext func(ctx context.Context, network, addr string) (addrs []string, err error)

// NewResolverMap creates resolver with predefined host -> IP map.
func NewResolverMap(hostToIP map[string][]string) ResolveContext {
	return func(ctx context.Context, network, addr string) ([]string, error) {
		addrHost, addrPort, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, &net.DNSError{Err: "invalid dial address", Name: addr, Server: "localmap", IsNotFound: true}
		}

		addrs := []string{addr}

		for _, addrIP := range FetchDNSFromCache(addrHost) {
			addrs = append(addrs, net.JoinHostPort(addrIP, addrPort))
		}

		for _, addrIP := range hostToIP[addrHost] {
			addrs = append(addrs, net.JoinHostPort(addrIP, addrPort))
		}

		return deduplicate(addrs), nil
	}
}

func deduplicate(list []string) (result []string) {
	m := make(map[string]struct{})

	for _, v := range list {
		if _, ok := m[v]; !ok {
			result = append(result, v)
			m[v] = struct{}{}
		}
	}

	return result
}
