package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	identity_handler "github.com/mysterium/node/cmd/mysterium_server/command_run/identity"
	"github.com/mysterium/node/communication"
	nats_dialog "github.com/mysterium/node/communication/nats/dialog"
	nats_discovery "github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/server/auth"
	openvpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"path/filepath"
)

func NewCommand(options CommandOptions) *CommandRun {
	return NewCommandWith(
		options,
		server.NewClient(),
		ip.NewResolver(),
		nat.NewService(),
	)
}

func NewCommandWith(
	options CommandOptions,
	mysteriumClient server.Client,
	ipResolver ip.Resolver,
	natService nat.NATService,
) *CommandRun {

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)
	cache := identity.NewIdentityCache(options.DirectoryKeystore, "remember.json")
	identityManager := identity.NewIdentityManager(keystoreInstance)
	createSigner := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(keystoreInstance, id)
	}
	identityHandler := identity_handler.NewHandler(
		identityManager,
		mysteriumClient,
		cache,
		createSigner,
	)

	// Country database downloaded from http://dev.maxmind.com/geoip/geoip2/geolite2/
	databasePath := filepath.Join(options.DirectoryConfig, "GeoLite2-Country.mmdb")
	locationDetector := location.NewDetector(databasePath)

	return &CommandRun{
		identityLoader: func() (identity.Identity, error) {
			return identity_handler.LoadIdentity(identityHandler, options.NodeKey, options.Passphrase)
		},
		createSigner:     createSigner,
		locationDetector: locationDetector,
		ipResolver:       ipResolver,
		mysteriumClient:  mysteriumClient,
		natService:       natService,
		dialogWaiterFactory: func(myID identity.Identity) communication.DialogWaiter {
			return nats_dialog.NewDialogWaiter(
				nats_discovery.NewAddressGenerate(myID),
				identity.NewSigner(keystoreInstance, myID),
			)
		},
		sessionManagerFactory: func(vpnServerIP string) session.Manager {
			return openvpn_session.NewManager(openvpn.NewClientConfig(
				vpnServerIP,
				filepath.Join(options.DirectoryConfig, "ca.crt"),
				filepath.Join(options.DirectoryConfig, "ta.key"),
			))
		},
		vpnServerFactory: func(manager session.Manager) *openvpn.Server {
			vpnServerConfig := openvpn.NewServerConfig(
				"10.8.0.0", "255.255.255.0",
				filepath.Join(options.DirectoryConfig, "ca.crt"),
				filepath.Join(options.DirectoryConfig, "server.crt"),
				filepath.Join(options.DirectoryConfig, "server.key"),
				filepath.Join(options.DirectoryConfig, "dh.pem"),
				filepath.Join(options.DirectoryConfig, "crl.pem"),
				filepath.Join(options.DirectoryConfig, "ta.key"),
			)
			sessionAuthenticator := openvpn_session.NewSessionAuthenticator(
				manager.FindSession,
				func(peerIdentity identity.Identity) identity.Verifier {
					return identity.NewVerifierIdentity(peerIdentity)
				},
			)
			vpnMiddlewares := []openvpn.ManagementMiddleware{
				auth.NewMiddleware(sessionAuthenticator.ValidateSession),
			}
			return openvpn.NewServer(vpnServerConfig, options.DirectoryRuntime, vpnMiddlewares...)
		},
	}
}
