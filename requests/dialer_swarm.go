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
	"time"
)

// DialerSwarm is a connects to multiple addresses in parallel and first successful connection wins.
type DialerSwarm struct {
	resolver *net.Resolver
	dialer   *net.Dialer

	// ResolveContext specifies the dial function for doing custom DNS lookup.
	// If ResolveContext is nil, then the transport dials using package net.
	ResolveContext func(ctx context.Context, network, host string) (addrs []string, err error)
}

// NewDialerSwarm creates swarm dialer with default configuration.
func NewDialerSwarm(srcIP string) *DialerSwarm {
	return &DialerSwarm{
		resolver: net.DefaultResolver,
		dialer: &net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(srcIP)},
		},
	}
}

// DialContext connects to the address on the named network using the provided context.
func (ds *DialerSwarm) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if ds.ResolveContext != nil {
		return ds.lookupAndDial(ctx, network, addr)
	}

	return ds.dialer.DialContext(ctx, network, addr)
}

func (ds *DialerSwarm) lookupAndDial(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	addrHost, addrPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
	}

	addrIPs, err := ds.ResolveContext(ctx, network, addrHost)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
	}

	for _, addrIP := range addrIPs {
		conn, err = ds.dialer.DialContext(ctx, network, net.JoinHostPort(addrIP, addrPort))
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Source: ds.dialer.LocalAddr, Addr: &net.IPAddr{IP: net.ParseIP(addrIP)}, Err: err}
		}
		break
	}

	return conn, nil
}
