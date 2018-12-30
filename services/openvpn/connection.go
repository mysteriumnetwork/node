/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package openvpn

import (
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/node/core/connection"
)

// OpenvpnStateMap maps openvpn states to connection state
var OpenvpnStateMap = map[openvpn.State]connection.State{
	openvpn.ConnectedState:    connection.Connected,
	openvpn.ExitingState:      connection.Disconnecting,
	openvpn.ReconnectingState: connection.Reconnecting,
}

// OpenVpnStateCallbackToConnectionState maps openvpn.State to connection.State. Returns a pointer to connection.state, or nil
func OpenVpnStateCallbackToConnectionState(input openvpn.State) connection.State {
	if val, ok := OpenvpnStateMap[input]; ok {
		return val
	}
	return connection.Unknown
}

// GetStateCallback returns the callback for working with openvpn state
func GetStateCallback(stateChannel connection.StateChannel) func(openvpnState openvpn.State) {
	return func(openvpnState openvpn.State) {
		connectionState := OpenVpnStateCallbackToConnectionState(openvpnState)
		if connectionState != connection.Unknown {
			stateChannel <- connectionState
		}

		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if openvpnState == openvpn.ProcessExited {
			close(stateChannel)
		}
	}
}
