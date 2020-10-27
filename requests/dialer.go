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
	"fmt"
	"net"
	"time"
)

// Dialer wraps default go dialer with extra features.
type Dialer struct {
	zeroDialer *net.Dialer
	addrToIP   map[string]string
}

// DialContext connects to the address on the named network using the provided context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if d.addrToIP != nil {
		_, addrPort, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid dial address: %w", err)
		}

		addrIP, exist := d.addrToIP[addr]
		if !exist {
			return nil, &net.DNSError{Err: "unmapped address", Name: addr, Server: "localmap", IsNotFound: true}
		}

		addr = addrIP + ":" + addrPort
	}

	return d.zeroDialer.DialContext(ctx, network, addr)
}

// NewDialer creates dialer with default configuration.
func NewDialer(srcIP string) *Dialer {
	return &Dialer{
		zeroDialer: &net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(srcIP)},
		},
	}
}

// NewDialerBypassDNS creates dialer which avoids DNS lookups.
func NewDialerBypassDNS(srcIP string, addrToIP map[string]string) *Dialer {
	dialer := NewDialer(srcIP)
	dialer.addrToIP = addrToIP

	return dialer
}
