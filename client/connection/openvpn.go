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
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	log "github.com/cihub/seelog"
	"errors"
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
		clientCountry, _ := detectCountry()
		statsSender := bytescount.NewSessionStatsSender(mysteriumAPIClient, vpnSession.ID, providerID, signer, clientCountry)
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

func detectCountry() (string, error) {
	ipResolver := ip.NewResolver("")
	locationDetector := location.NewDetector("")

	ip, err := ipResolver.GetPublicIP()
	if err != nil {
		return "", errors.New("IP detection failed: " + err.Error())
	}

	country, err := locationDetector.DetectCountry(ip)
	if err != nil {
		return "", errors.New("Country detection failed: " + err.Error())
	}

	log.Info("Country detected: ", country)
	return country, nil
}