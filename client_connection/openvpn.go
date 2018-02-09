package client_connection

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/openvpn/middlewares/client/bytescount"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	openvpnSession "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"path/filepath"
	"time"
)

// ConfigureVpnClientFactory creates openvpn construction function by given vpn session, consumer id and state callbacks
func ConfigureVpnClientFactory(
	mysteriumAPIClient server.Client,
	configDirectory, runtimeDirectory string,
	signerFactory identity.SignerFactory,
	statsKeeper bytescount.SessionStatsKeeper,
) VpnClientCreator {
	return func(vpnSession session.SessionDto, consumerID identity.Identity, stateCallback state.ClientStateCallback) (openvpn.Client, error) {
		vpnClientConfig, err := openvpn.NewClientConfigFromString(
			vpnSession.Config,
			filepath.Join(runtimeDirectory, "client.ovpn"),
			filepath.Join(configDirectory, "update-resolv-conf"),
			filepath.Join(configDirectory, "update-resolv-conf"),
		)
		if err != nil {
			return nil, err
		}

		signer := signerFactory(consumerID)

		statsSaver := bytescount.NewSessionStatsSaver(statsKeeper)
		statsSender := bytescount.NewSessionStatsSender(mysteriumAPIClient, vpnSession.ID, signer)
		statsHandler := bytescount.NewCompositeStatsHandler(statsSaver, statsSender)

		credentialsProvider := openvpnSession.SignatureCredentialsProvider(vpnSession.ID, signer)

		return openvpn.NewClient(
			vpnClientConfig,
			runtimeDirectory,
			state.NewMiddleware(stateCallback),
			bytescount.NewMiddleware(statsHandler, 1*time.Minute),
			auth.NewMiddleware(credentialsProvider),
		), nil
	}
}
