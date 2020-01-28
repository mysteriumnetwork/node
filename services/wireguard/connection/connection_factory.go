/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services/wireguard/key"
	"github.com/pkg/errors"
)

// Factory is the wireguard connection factory
type Factory struct {
	configDir  string
	ipResolver ip.Resolver
	natPinger  traversal.NATProviderPinger
}

// Create creates a new wireguard connection
func (f *Factory) Create(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
	privateKey, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate private key")
	}

	return &Connection{
		done:              make(chan struct{}),
		statsCheckerStop:  make(chan struct{}),
		pingerStop:        make(chan struct{}),
		stateChannel:      stateChannel,
		statisticsChannel: statisticsChannel,
		privateKey:        privateKey,
		configDir:         f.configDir,
		ipResolver:        f.ipResolver,
		natPinger:         f.natPinger,
	}, nil
}

// NewConnectionCreator creates wireguard connections
func NewConnectionCreator(configDir string, ipResolver ip.Resolver, natPinger traversal.NATProviderPinger) connection.Factory {
	return &Factory{
		configDir:  configDir,
		ipResolver: ipResolver,
		natPinger:  natPinger,
	}
}
