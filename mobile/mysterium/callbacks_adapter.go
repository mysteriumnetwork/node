package mysterium

import (
	"github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn3"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/connection"
)

type StatsUpdater interface {
	Save(stats stats.SessionStats)
}

func channelToCallbacks(channel connection.StateChannel, updater StatsUpdater) openvpn3.MobileSessionCallbacks {

	return channelToCallbacksAdapter{
		channel:      channel,
		statsUpdater: updater,
	}
}

type channelToCallbacksAdapter struct {
	channel      connection.StateChannel
	statsUpdater StatsUpdater
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
