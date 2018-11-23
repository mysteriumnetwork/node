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
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/session"
)

type sessionWrapper struct {
	session *openvpn3.Session
}

func (wrapper *sessionWrapper) Start() error {

	wrapper.session.Start()
	return nil
}

func (wrapper *sessionWrapper) Stop() {
	wrapper.session.Stop()
}

func (wrapper *sessionWrapper) Wait() error {
	return wrapper.session.Wait()
}

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
	adapter.statsUpdater.Save(stats.SessionStats{
		BytesSent:     uint64(openvpnStats.BytesOut),
		BytesReceived: uint64(openvpnStats.BytesIn),
	})
}

// Openvpn3TunnelSetup is alias for openvpn3 tunnel setup interface exposed to Android/iOS interop
type Openvpn3TunnelSetup openvpn3.TunnelSetup

// OverrideOpenvpnConnection replaces default openvpn connection factory with mobile related one
func (mobNode *MobileNode) OverrideOpenvpnConnection(tunnelSetup Openvpn3TunnelSetup) {
	mobNode.di.ConnectionRegistry.Register("openvpn", func(options connection.ConnectOptions, channel connection.StateChannel) (connection.Connection, error) {

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(options.SessionConfig, "", "")
		if err != nil {
			return nil, err
		}

		profileContent, err := vpnClientConfig.ToConfigFileContent()
		if err != nil {
			return nil, err
		}

		config := openvpn3.NewConfig(profileContent)
		config.GuiVersion = "govpn 0.1"
		config.CompressionMode = "asym"
		config.ConnTimeout = 0 //essentially means - reconnect forever (but can always stopped with disconnect)

		signer := mobNode.di.SignerFactory(options.ConsumerID)

		username, password, err := session.SignatureCredentialsProvider(options.SessionID, signer)()
		if err != nil {
			return nil, err
		}

		credentials := openvpn3.UserCredentials{
			Username: username,
			Password: password,
		}

		session := openvpn3.NewMobileSession(config, credentials, channelToCallbacks(channel, mobNode.di.StatsKeeper), tunnelSetup)

		return &sessionWrapper{
			session: session,
		}, nil
	})
}
