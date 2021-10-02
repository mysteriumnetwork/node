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

	"github.com/mysteriumnetwork/node/requests/resolver"
)

// DialContext specifies the dial function for creating unencrypted TCP connections.
type DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

// Dialer wraps default go dialer with extra features.
type Dialer struct {
	// ResolveContext specifies the resolve function for doing custom DNS lookup.
	// If ResolveContext is nil, then the transport dials using package net.
	ResolveContext resolver.ResolveContext

	// Dialer specifies the dial function for creating unencrypted TCP connections.
	Dialer DialContext
}

// NewDialer creates dialer with default configuration.
func NewDialer(srcIP string) *Dialer {
	return &Dialer{
		Dialer: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(srcIP)},
		}).DialContext,
	}
}

// DialContext connects to the address on the named network using the provided context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	if d.ResolveContext != nil {
		addrs, err := d.ResolveContext(ctx, network, addr)
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
		}

		conn, err := d.dialAddrs(ctx, network, addrs)
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
		}

		return conn, nil
	}

	return d.Dialer(ctx, network, addr)
}

func (d *Dialer) dialAddrs(ctx context.Context, network string, addrs []string) (conn net.Conn, err error) {
	for _, addr := range addrs {
		conn, err = d.Dialer(ctx, network, addr)
		if err == nil {
			return conn, nil
		}
	}

	return conn, err
}
