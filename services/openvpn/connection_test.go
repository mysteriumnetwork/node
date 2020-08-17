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

package openvpn

import (
	"reflect"
	"testing"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
)

func TestGetStateCallbackReturnsCorrectState(t *testing.T) {
	channel := make(chan connectionstate.State, 1)
	callback := getStateCallback(channel)
	callback(openvpn.ConnectedState)
	assert.Equal(t, connectionstate.Connected, <-channel)
}

func TestGetStateCallbackClosesChannelOnProcessExit(t *testing.T) {
	channel := make(chan connectionstate.State, 1)
	callback := getStateCallback(channel)
	callback(openvpn.ExitingState)
	res, ok := <-channel
	assert.Equal(t, connectionstate.Disconnecting, res)
	assert.True(t, ok)
}

func TestOpenVpnStateCallbackToConnectionState(t *testing.T) {
	type args struct {
		input openvpn.State
	}
	tests := []struct {
		name string
		args args
		want connectionstate.State
	}{
		{
			name: "Maps openvpn.connectedState to connection.Connected",
			args: args{
				input: openvpn.ConnectedState,
			},
			want: connectionstate.Connected,
		},
		{
			name: "Maps openvpn.exitingState to connection.Disconnecting",
			args: args{
				input: openvpn.ExitingState,
			},
			want: connectionstate.Disconnecting,
		},
		{
			name: "Maps openvpn.reconnectingState to connection.Reconnecting",
			args: args{
				input: openvpn.ReconnectingState,
			},
			want: connectionstate.Reconnecting,
		},
		{
			name: "Maps openvpn.getConfigState to connection.Unknown",
			args: args{
				input: openvpn.GetConfigState,
			},
			want: connectionstate.Unknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openVpnStateCallbackToConnectionState(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OpenVpnStateCallbackToConnectionState() = %v, want %v", got, tt.want)
			}
		})
	}
}
