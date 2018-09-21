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
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/connection"
)

func TestGetStateCallbackReturnsCorrectState(t *testing.T) {
	statsKeeper := &fakeSessionStatsKeeper{}
	channel := make(chan connection.State, 1)
	callback := GetStateCallback(channel, statsKeeper)
	callback(openvpn.ConnectedState)
	assert.Equal(t, connection.Connected, <-channel)
	assert.True(t, statsKeeper.sessionStartMarked)
}

func TestGetStateCallbackClosesChannelOnProcessExit(t *testing.T) {
	statsKeeper := &fakeSessionStatsKeeper{}
	channel := make(chan connection.State, 1)
	callback := GetStateCallback(channel, statsKeeper)
	callback(openvpn.ExitingState)
	res, ok := <-channel
	assert.Equal(t, connection.Disconnecting, res)
	assert.True(t, ok)
	assert.True(t, statsKeeper.sessionEndMarked)
}

func TestOpenVpnStateCallbackToConnectionState(t *testing.T) {
	type args struct {
		input openvpn.State
	}
	tests := []struct {
		name string
		args args
		want connection.State
	}{
		{
			name: "Maps openvpn.ConnectedState to connection.Connected",
			args: args{
				input: openvpn.ConnectedState,
			},
			want: connection.Connected,
		},
		{
			name: "Maps openvpn.ExitingState to connection.Disconnecting",
			args: args{
				input: openvpn.ExitingState,
			},
			want: connection.Disconnecting,
		},
		{
			name: "Maps openvpn.ReconnectingState to connection.Reconnecting",
			args: args{
				input: openvpn.ReconnectingState,
			},
			want: connection.Reconnecting,
		},
		{
			name: "Maps openvpn.GetConfigState to connection.Unknown",
			args: args{
				input: openvpn.GetConfigState,
			},
			want: connection.Unknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OpenVpnStateCallbackToConnectionState(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OpenVpnStateCallbackToConnectionState() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeSessionStatsKeeper struct {
	sessionStartMarked, sessionEndMarked bool
}

func (fsk *fakeSessionStatsKeeper) Save(stats stats.SessionStats) {
}

func (fsk *fakeSessionStatsKeeper) Retrieve() stats.SessionStats {
	return stats.SessionStats{}
}

func (fsk *fakeSessionStatsKeeper) MarkSessionStart() {
	fsk.sessionStartMarked = true
}

func (fsk *fakeSessionStatsKeeper) GetSessionDuration() time.Duration {
	return time.Duration(0)
}

func (fsk *fakeSessionStatsKeeper) MarkSessionEnd() {
	fsk.sessionEndMarked = true
}
