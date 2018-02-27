package server

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	identity_handler "github.com/mysterium/node/cmd/commands/server/identity"
	"github.com/mysterium/node/communication"
	nats_dialog "github.com/mysterium/node/communication/nats/dialog"
	nats_discovery "github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/server/auth"
	"github.com/mysterium/node/openvpn/middlewares/state"
	openvpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"path/filepath"
)

// NewCommand function creates new server command by given options
func NewCommand(options CommandOptions) *Command {
	return NewCommandWith(
		options,
		server.NewClient(),
		ip.NewResolver(),
		nat.NewService(),
	)
}

// NewCommandWith function creates new client command by given options + injects given dependencies
func NewCommandWith(
	options CommandOptions,
	mysteriumClient server.Client,
	ipResolver ip.Resolver,
	natService nat.NATService,
) *Command {

	keystoreDirectory := filepath.Join(options.DirectoryData, "keystore")
	keystoreInstance := keystore.NewKeyStore(keystoreDirectory, keystore.StandardScryptN, keystore.StandardScryptP)
	createSigner := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(keystoreInstance, id)
	}

	identityHandler := identity_handler.NewHandler(
		identity.NewIdentityManager(keystoreInstance),
		mysteriumClient,
		identity.NewIdentityCache(keystoreDirectory, "remember.json"),
		createSigner,
	)

	var locationDetector location.Detector
	if options.LocationCountry != "" {
		locationDetector = location.NewDetectorFake(options.LocationCountry)
	} else if options.LocationDatabase != "" {
		locationDetector = location.NewDetector(filepath.Join(options.DirectoryConfig, options.LocationDatabase))
	} else {
		locationDetector = location.NewDetector(filepath.Join(options.DirectoryConfig, defaultLocationDatabase))
	}

	return &Command{
		identityLoader: func() (identity.Identity, error) {
			return identity_handler.LoadIdentity(identityHandler, options.Identity, options.Passphrase)
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
			clientConfigGenerator := openvpn.NewClientConfigGenerator(options.DirectoryRuntime, vpnServerIP)

			return openvpn_session.NewManager(
				clientConfigGenerator,
				&session.UUIDGenerator{},
			)
		},
		vpnServerFactory: func(manager session.Manager, serviceLocation dto.Location, providerID identity.Identity, callback state.Callback) *openvpn.Server {
			serverConfigGenerator := openvpn.NewServerConfigGenerator(options.DirectoryRuntime, serviceLocation, providerID)
			sessionValidator := openvpn_session.NewSessionValidator(
				manager.FindSession,
				identity.NewExtractor(),
			)

			return openvpn.NewServer(
				options.OpenvpnBinary,
				serverConfigGenerator,
				options.DirectoryRuntime,
				auth.NewMiddleware(sessionValidator),
				state.NewMiddleware(callback),
			)
		},
		checkOpenvpn: func() error {
			return openvpn.CheckOpenvpnBinary(options.OpenvpnBinary)
		},
	}
}
