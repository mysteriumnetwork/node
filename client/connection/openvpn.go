package connection

import (
	"github.com/mysterium/node/identity"
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
) VpnClientCreator {
	return func(vpnSession session.SessionDto, consumerID identity.Identity, providerID identity.Identity, stateCallback state.Callback) (openvpn.Client, error) {
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
		statsSender := bytescount.NewSessionStatsSender(mysteriumAPIClient, vpnSession.ID, providerID, signer)
		asyncStatsSender := func(stats bytescount.SessionStats) error {
			go statsSender(stats)
			return nil
		}

		statsHandler := bytescount.NewCompositeStatsHandler(statsSaver, asyncStatsSender)

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
