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

package nat

import (
	"context"
	"net"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/traversal"
)

// StartPorts starts the process of serving connections for the provided ports.
type StartPorts func(ctx context.Context, peerIP string, peerPorts, localPorts []int) ([]*net.UDPConn, error)

type natHolePunchingPort struct {
	pool   *port.Pool
	pinger traversal.NATPinger
}

// NewNATHolePunchingPortProvider creates new instance of the NAT hole punching port provider.
func NewNATHolePunchingPortProvider() PortProvider {
	udpPortRange, err := port.ParseRange(config.GetString(config.FlagUDPListenPorts))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse UDP listen port range, using default value")

		udpPortRange, err = port.ParseRange("10000:60000")
		if err != nil {
			panic(err) // This must never happen.
		}
	}

	return &natHolePunchingPort{
		pool:   port.NewFixedRangePool(udpPortRange),
		pinger: traversal.NewPinger(traversal.DefaultPingConfig(), eventbus.New()),
	}
}

func (hp *natHolePunchingPort) PreparePorts() (ports []int, release func(), start StartPorts, err error) {
	poolPorts, err := hp.pool.AcquireMultiple(pingMaxPorts)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, p := range poolPorts {
		ports = append(ports, p.Num())
	}

	return ports, func() {}, hp.Start, nil
}

func (hp *natHolePunchingPort) Start(ctx context.Context, peerIP string, peerPorts, localPorts []int) ([]*net.UDPConn, error) {
	return hp.pinger.PingConsumerPeer(context.Background(), "remove this id", peerIP, localPorts, peerPorts, providerInitialTTL, requiredConnCount)
}
