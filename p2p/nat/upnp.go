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
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/nat/mapping"
)

type upnpPort struct {
	pool       *port.Pool
	portMapper mapping.PortMapper
}

// NewUPnPPortProvider returns a new instance of the UPnP port provider.
func NewUPnPPortProvider() PortProvider {
	udpPortRange, err := port.ParseRange(config.GetString(config.FlagUDPListenPorts))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse UDP listen port range, using default value")

		udpPortRange, err = port.ParseRange("10000:60000")
		if err != nil {
			panic(err) // This must never happen.
		}
	}

	return &upnpPort{
		pool:       port.NewFixedRangePool(udpPortRange),
		portMapper: mapping.NewPortMapper(mapping.DefaultConfig(), eventbus.New()),
	}
}

func (up *upnpPort) PreparePorts() (ports []int, release func(), start StartPorts, err error) {
	localPorts, err := up.pool.AcquireMultiple(requiredConnCount)
	if err != nil {
		return nil, nil, nil, err
	}

	// Try to add upnp ports mapping.
	var portsRelease []func()
	var portMappingOk bool
	var portRelease func()

	for _, p := range localPorts {
		portRelease, portMappingOk = up.portMapper.Map("", "UDP", p.Num(), "Myst node p2p port mapping")
		if !portMappingOk {
			break
		}

		portsRelease = append(portsRelease, portRelease)
	}

	if !portMappingOk {
		for _, r := range portsRelease {
			r()
		}
		return nil, nil, nil, fmt.Errorf("failed to map port via UPnP")
	}

	for _, p := range localPorts {
		ports = append(ports, p.Num())
	}

	if err := checkAllPorts(ports); err != nil {
		for _, r := range portsRelease {
			r()
		}
		log.Debug().Err(err).Msgf("Failed to check UPnP ports %d globally", ports)
		return nil, nil, nil, err
	}

	return ports, func() {
		for _, r := range portsRelease {
			r()
		}
	}, nil, nil
}
