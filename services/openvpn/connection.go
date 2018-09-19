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
	"encoding/json"
	"time"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/auth"
	openvpn_bytescount "github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/client/bytescount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/client/stats"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/services/openvpn/middlewares/client/bytescount"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
)

// ConfigureOpenVpnClientFactory creates openvpn construction function by given vpn session, consumer id and state callbacks
func ConfigureOpenVpnClientFactory(
	mysteriumAPIClient server.Client,
	openvpnBinary, configDirectory, runtimeDirectory string,
	signerFactory identity.SignerFactory,
	statsKeeper stats.SessionStatsKeeper,
	originalLocationCache location.Cache,
) connection.VpnConnectionCreator {
	return func(connectionOptions connection.ConnectOptions, stateChannel connection.StateChannel) (connection.Connection, error) {
		var receivedConfig VPNConfig
		err := json.Unmarshal(connectionOptions.Config, &receivedConfig)
		if err != nil {
			return nil, err
		}

		vpnClientConfig, err := NewClientConfigFromSession(&receivedConfig, configDirectory, runtimeDirectory)
		if err != nil {
			return nil, err
		}

		signer := signerFactory(connectionOptions.ConsumerID)

		statsSaver := bytescount.NewSessionStatsSaver(statsKeeper)

		statsSender := stats.NewRemoteStatsSender(
			statsKeeper,
			mysteriumAPIClient,
			connectionOptions.SessionID,
			connectionOptions.ProviderID,
			signer,
			originalLocationCache.Get().Country,
			time.Minute,
		)
		credentialsProvider := openvpn_session.SignatureCredentialsProvider(connectionOptions.SessionID, signer)

		openvpnStateCallback := func(openvpnState openvpn.State) {
			connectionState := OpenVpnStateCallbackToConnectionState(openvpnState)
			if connectionState != connection.Unknown {
				stateChannel <- connectionState
			}

			//this is the last state - close channel (according to best practices of go - channel writer controls channel)
			if openvpnState == openvpn.ProcessExited {
				close(stateChannel)
			}
		}

		return NewClient(
			openvpnBinary,
			vpnClientConfig,
			state.NewMiddleware(openvpnStateCallback, statsSender.StateHandler),
			openvpn_bytescount.NewMiddleware(statsSaver, 1*time.Second),
			auth.NewMiddleware(credentialsProvider),
		), nil
	}
}

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
