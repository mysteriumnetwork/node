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
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
)

type manualPort struct {
	pool *port.Pool
}

// NewManualPortProvider creates new instance of the manual port provider.
func NewManualPortProvider() PortProvider {
	udpPortRange, err := port.ParseRange(config.GetString(config.FlagUDPListenPorts))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse UDP listen port range, using default value")

		udpPortRange, err = port.ParseRange("10000:60000")
		if err != nil {
			panic(err) // This must never happen.
		}
	}

	return &manualPort{port.NewFixedRangePool(udpPortRange)}
}

func (mp *manualPort) PreparePorts() (ports []int, release func(), start StartPorts, err error) {
	poolPorts, err := mp.pool.AcquireMultiple(requiredConnCount)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, p := range poolPorts {
		ports = append(ports, p.Num())
	}

	if err := checkAllPorts(ports); err != nil {
		log.Debug().Err(err).Msgf("Failed to check manual ports %d globally", ports)
		return nil, nil, nil, err
	}

	return ports, nil, nil, nil
}
