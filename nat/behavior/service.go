/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package behavior

import (
	"context"
	"time"
)

// Only servers compatible with RFC 5780 are usable!
var compatibleSTUNServers = []string{
	"stun.mysterium.network:3478",
	"stun.stunprotocol.org:3478",
	"stun.sip.us:3478",
}

const concurrentRequestTimeout = 1 * time.Second

// NATProber is an abstaction over instances capable probing NAT and
// returning either it's type or error.
type NATProber interface {
	Probe(context.Context) (string, error)
}

// NewNATProber constructs some suitable NATProber without any implementation
// guarantees.
func NewNATProber() NATProber {
	return newConcurrentNATProber(compatibleSTUNServers, concurrentRequestTimeout)
}

type concurrentNATProber struct {
	servers []string
	timeout time.Duration
}

func newConcurrentNATProber(servers []string, timeout time.Duration) *concurrentNATProber {
	return &concurrentNATProber{
		servers: servers,
		timeout: timeout,
	}
}

func (p *concurrentNATProber) Probe(ctx context.Context) (string, error) {
	return RacingDiscoverNATBehavior(ctx, p.servers, p.timeout)
}
