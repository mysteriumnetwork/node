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
	"github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn3"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/connection"
)

type statsUpdater interface {
	Save(stats stats.SessionStats)
}

func channelToCallbacks(channel connection.StateChannel, updater statsUpdater) openvpn3.MobileSessionCallbacks {

	return channelToCallbacksAdapter{
		channel:      channel,
		statsUpdater: updater,
	}
}

type channelToCallbacksAdapter struct {
	channel      connection.StateChannel
	statsUpdater statsUpdater
}

func (adapter channelToCallbacksAdapter) OnEvent(event openvpn3.Event) {

	switch event.Name {
	case "CONNECTING":
		adapter.channel <- connection.Connecting
	case "CONNECTED":
		adapter.channel <- connection.Connected
	case "DISCONNECTED":
		adapter.channel <- connection.NotConnected
		close(adapter.channel)
	default:
		seelog.Infof("Unhandled event: %+v", event)
	}
}

func (channelToCallbacksAdapter) Log(text string) {
	seelog.Infof("Log: %+v", text)
}

func (adapter channelToCallbacksAdapter) OnStats(openvpnStats openvpn3.Statistics) {
	seelog.Infof("Stats: %+v", openvpnStats)

	adapter.statsUpdater.Save(stats.SessionStats{
		BytesSent:     uint64(openvpnStats.BytesOut),
		BytesReceived: uint64(openvpnStats.BytesIn),
	})
}
