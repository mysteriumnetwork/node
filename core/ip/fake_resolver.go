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

import "net"

// NewResolverMock returns mockResolver which resolves statically entered IP
func NewResolverMock(ipAddress string) Resolver {
	return &mockResolver{
		ipAddress: ipAddress,
		error:     nil,
	}
}

// NewResolverMockFailing returns mockResolver with entered error
func NewResolverMockFailing(err error) Resolver {
	return &mockResolver{
		ipAddress: "",
		error:     err,
	}
}

type mockResolver struct {
	ipAddress string
	error     error
}

func (client *mockResolver) GetPublicIP() (string, error) {
	return client.ipAddress, client.error
}

func (client *mockResolver) GetOutboundIP() (net.IP, error) {
	ipAddress := net.ParseIP(client.ipAddress)
	localIPAddress := net.UDPAddr{IP: ipAddress}
	return localIPAddress.IP, client.error
}

func (client *mockResolver) GetOutboundIPAsString() (string, error) {
	return client.ipAddress, client.error
}
