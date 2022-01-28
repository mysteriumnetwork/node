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
	"errors"
	"time"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat"
)

// Only servers compatible with RFC 5780 are usable!
var compatibleSTUNServers = []string{
	"stun.mysterium.network:3478",
	"stun.stunprotocol.org:3478",
	"stun.sip.us:3478",
}

const (
	// AppTopicNATTypeDetected represents NAT type detection topic.
	AppTopicNATTypeDetected = "NAT-type-detected"

	concurrentRequestTimeout = 1 * time.Second
)

// ErrInappropriateState error is returned by gatedNATProber when connection
// is active
var ErrInappropriateState = errors.New("NAT probing is impossible at this connection state")

// NATProber is an abstaction over instances capable probing NAT and
// returning either it's type or error.
type NATProber interface {
	Probe(context.Context) (nat.NATType, error)
}

// ConnectionStatusProvider is a subset of connection.Manager methods
// to provide gatedNATProber with current connection status
type ConnectionStatusProvider interface {
	Status(int) connectionstate.Status
}

// NewNATProber constructs some suitable NATProber without any implementation
// guarantees.
func NewNATProber(connStatusProvider ConnectionStatusProvider, eventbus eventbus.Publisher) NATProber {
	var prober NATProber
	prober = newConcurrentNATProber(compatibleSTUNServers, concurrentRequestTimeout)
	prober = newGatedNATProber(connStatusProvider, eventbus, prober)
	return prober
}

// Probes NAT status with parallel tests against multiple STUN servers
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

func (p *concurrentNATProber) Probe(ctx context.Context) (nat.NATType, error) {
	return RacingDiscoverNATBehavior(ctx, p.servers, p.timeout)
}

// Gates calls to other NATProber, allowing them only when node is not connected
type gatedNATProber struct {
	next               NATProber
	connStatusProvider ConnectionStatusProvider
	eventbus           eventbus.Publisher
}

func newGatedNATProber(connStatusProvider ConnectionStatusProvider, eventbus eventbus.Publisher, next NATProber) *gatedNATProber {
	return &gatedNATProber{
		next:               next,
		connStatusProvider: connStatusProvider,
		eventbus:           eventbus,
	}
}

func (p *gatedNATProber) Probe(ctx context.Context) (nat.NATType, error) {
	if p.connStatusProvider.Status(0).State != connectionstate.NotConnected {
		return "", ErrInappropriateState
	}

	s, err := p.next.Probe(ctx)
	if err != nil {
		return "", err
	}

	p.eventbus.Publish(AppTopicNATTypeDetected, s)
	return s, nil
}
