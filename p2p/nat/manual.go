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

func NewManualPortProvider() PortProvider {
	udpPortRange, err := port.ParseRange(config.GetString(config.FlagUDPListenPorts))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse UDP listen port range, using default value")
		// return port.Range{}, fmt.Errorf("failed to parse UDP ports: %w", err)
	}

	return &manualPort{port.NewFixedRangePool(*udpPortRange)}
}

func (mp *manualPort) PreparePorts() (ports []int, release func(), start StartPorts, err error) {
	poolPorts, err := mp.pool.AcquireMultiple(2)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, p := range poolPorts {
		ports = append(ports, p.Num())
	}

	return ports, func() {}, nil, nil
}
