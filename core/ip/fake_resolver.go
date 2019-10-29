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

package ip

import (
	"fmt"
	"net"
)

// NewResolverMock returns mockResolver which resolves statically entered IP.
// If multiple addresses are provided then return will depend on call order.
func NewResolverMock(ipAddresses ...string) Resolver {
	return &mockResolver{
		ipAddresses: ipAddresses,
		error:       nil,
	}
}

// NewResolverMockFailing returns mockResolver with entered error
func NewResolverMockFailing(err error) Resolver {
	return &mockResolver{
		ipAddresses: []string{""},
		error:       err,
	}
}

type mockResolver struct {
	ipAddresses []string
	error       error
}

func (client *mockResolver) GetPublicIP() (string, error) {
	return client.getNextIP(), client.error
}

func (client *mockResolver) GetOutboundIP() (net.IP, error) {
	ipAddress := net.ParseIP(client.getNextIP())
	localIPAddress := net.UDPAddr{IP: ipAddress}
	return localIPAddress.IP, client.error
}

func (client *mockResolver) GetOutboundIPAsString() (string, error) {
	return client.getNextIP(), client.error
}

func (client *mockResolver) getNextIP() string {
	// Return first address if only one provided.
	if len(client.ipAddresses) == 1 {
		fmt.Println("get first ip")
		return client.ipAddresses[0]
	}
	// Return first address and dequeue from address list. This allows to
	// mock to return different value for each call.
	if len(client.ipAddresses) > 0 {
		ip := client.ipAddresses[0]
		client.ipAddresses = client.ipAddresses[1:]
		return ip
	}
	return ""
}
