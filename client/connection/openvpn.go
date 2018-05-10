package connection

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/state"
	openvpnSession "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"path/filepath"
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
	return func(vpnSession session.SessionDto, consumerID identity.Identity, providerID identity.Identity, stateCallback state.Callback) (openvpn.Client, error) {

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(vpnSession.Config, configDirectory, filepath.Join(runtimeDirectory, "client.ovpn"))
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

		credentialsProvider := openvpnSession.SignatureCredentialsProvider(vpnSession.ID, signer)

		return openvpn.NewClient(
			openvpnBinary,
			vpnClientConfig,
			runtimeDirectory,
			state.NewMiddleware(stateCallback),
			bytescount.NewMiddleware(statsHandler, 1*time.Second),
			auth.NewMiddleware(credentialsProvider),
		), nil
	}
}

func channelToStateCallbackAdapter(channel chan openvpn.State) state.Callback {
	return func(state openvpn.State) {
		channel <- state
		if state == openvpn.ExitingState {
			//this is the last state - close channel (according to best practices of go - channel writer controls channel)
			close(channel)
		}
	}
}
