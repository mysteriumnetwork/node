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
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	endpoint "github.com/mysteriumnetwork/node/services/wireguard/endpoint"
)

// NewConnectionCreator creates wireguard connections
func NewConnectionCreator() connection.Factory {
	return func(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
		privateKey, err := endpoint.GeneratePrivateKey()
		if err != nil {
			return nil, err
		}
		config := wg.ServiceConfig{}
		config.Consumer.PrivateKey = privateKey

		return &Connection{
			stateChannel: stateChannel,
			config:       config,
		}, nil
	}
}
