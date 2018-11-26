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

package mysterium

import (
	"errors"

	"github.com/mysteriumnetwork/node/core/connection"
)

// TODO this probably can be aligned with openvpn3 setup interface as its very close to android api, but for sake of
// package independencies we are not reusing the same interface although implementation is rougly the same
type tunnSetupPlaceholder interface {
	Establish() int
}

// WireguardTunnelSetup represents interface for setuping tunnel on caller side (i.e. android api)
type WireguardTunnelSetup tunnSetupPlaceholder

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one
func (mobNode *MobileNode) OverrideWireguardConnection(wgTunnelSetup WireguardTunnelSetup) {
	mobNode.di.ConnectionRegistry.Register("wireguard", func(options connection.ConnectOptions, stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {
		return nil, errors.New("not implemented yet")
	})
}
