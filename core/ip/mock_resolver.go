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
	"net"
)

// NewResolverMock returns mockResolver which resolves statically entered IP.
// If multiple addresses are provided then return will depend on call order.
func NewResolverMock(ip string) Resolver {
	return &mockResolver{
		publicIP:   ip,
		outboundIP: net.ParseIP(ip),
		error:      nil,
	}
}

// NewResolverMockMultiple returns mockResolver which resolves statically entered IP.
// If multiple addresses are provided then return will depend on call order.
func NewResolverMockMultiple(outboundIP string, publicIPs ...string) Resolver {
	return &mockResolver{
		publicIPs:  publicIPs,
		outboundIP: net.ParseIP(outboundIP),
		error:      nil,
	}
}

// NewResolverMockFailing returns mockResolver with entered error
func NewResolverMockFailing(err error) Resolver {
	return &mockResolver{
		error: err,
	}
}

type mockResolver struct {
	publicIP   string
	publicIPs  []string
	outboundIP net.IP
	error      error
}

func (client *mockResolver) MockPublicIPs(ips ...string) {
	client.publicIPs = ips
}

func (client *mockResolver) GetPublicIP() (string, error) {
	if client.publicIPs != nil {
		return client.getNextIP(), client.error
	}
	return client.publicIP, client.error
}

func (client *mockResolver) GetProxyIP(_ int) (string, error) {
	if client.publicIPs != nil {
		return client.getNextIP(), client.error
	}
	return client.publicIP, client.error
}

func (client *mockResolver) GetOutboundIP() (string, error) {
	return client.outboundIP.String(), client.error
}

func (client *mockResolver) getNextIP() string {
	// Return first address if only one provided.
	if len(client.publicIPs) == 1 {
		return client.publicIPs[0]
	}
	// Return first address and dequeue from address list. This allows to
	// mock to return different value for each call.
	if len(client.publicIPs) > 0 {
		ip := client.publicIPs[0]
		client.publicIPs = client.publicIPs[1:]
		return ip
	}
	return ""
}
