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

package connection

import (
	"encoding/json"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/openvpn/session/credentials"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"time"
)

// ConfigureVpnClientFactory creates openvpn construction function by given vpn session, consumer id and state callbacks
func ConfigureVpnClientFactory(
	mysteriumAPIClient server.Client,
	openvpnBinary, configDirectory, runtimeDirectory string,
	signerFactory identity.SignerFactory,
	statsKeeper bytescount.SessionStatsKeeper,
	originalLocationCache location.Cache,
) VpnClientCreator {
	return func(vpnSession session.SessionDto, consumerID identity.Identity, providerID identity.Identity, stateCallback state.Callback) (openvpn.Process, error) {
		var receivedConfig openvpn.VPNConfig
		err := json.Unmarshal(vpnSession.Config, &receivedConfig)
		if err != nil {
			return nil, err
		}

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(&receivedConfig, configDirectory, runtimeDirectory)
		if err != nil {
			return nil, err
		}

		signer := signerFactory(consumerID)

		statsSaver := bytescount.NewSessionStatsSaver(statsKeeper)

		originalLocation := originalLocationCache.Get()

		statsSender := bytescount.NewSessionStatsSender(
			mysteriumAPIClient,
			vpnSession.ID,
			providerID,
			signer,
			originalLocation.Country,
		)
		asyncStatsSender := func(stats bytescount.SessionStats) error {
			go statsSender(stats)
			return nil
		}
		intervalStatsSender, err := bytescount.NewIntervalStatsHandler(asyncStatsSender, time.Now, time.Minute)
		if err != nil {
			return nil, err
		}
		statsHandler := bytescount.NewCompositeStatsHandler(statsSaver, intervalStatsSender)

		credentialsProvider := credentials.SignatureCredentialsProvider(vpnSession.ID, signer)

		return openvpn.NewClient(
			openvpnBinary,
			vpnClientConfig,
			state.NewMiddleware(stateCallback),
			bytescount.NewMiddleware(statsHandler, 1*time.Second),
			auth.NewMiddleware(credentialsProvider),
		), nil
	}
}

func channelToStateCallbackAdapter(channel chan openvpn.State) state.Callback {
	return func(state openvpn.State) {
		channel <- state
		if state == openvpn.ProcessExited {
			//this is the last state - close channel (according to best practices of go - channel writer controls channel)
			close(channel)
		}
	}
}
