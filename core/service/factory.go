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

package service

import (
	"path/filepath"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	identity_handler "github.com/mysteriumnetwork/node/cmd/commands/service/identity"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/ip"
	"github.com/mysteriumnetwork/node/location"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/openvpn"
	"github.com/mysteriumnetwork/node/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/node/openvpn/middlewares/state"
	openvpn_session "github.com/mysteriumnetwork/node/openvpn/session"
	"github.com/mysteriumnetwork/node/openvpn/tls"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/session"
)

// NewManager function creates new service manager by given options
func NewManager(nodeOptions node.Options, serviceOptions Options) *Manager {

	networkDefinition := node.GetNetworkDefinition(nodeOptions.NetworkOptions)
	mysteriumClient := server.NewClient(networkDefinition.DiscoveryAPIAddress)

	logconfig.Bootstrap()

	ipResolver := ip.NewResolver(nodeOptions.IpifyUrl)
	natService := nat.NewService()

	keystoreDirectory := filepath.Join(nodeOptions.Directories.Data, "keystore")
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

	var locationResolver location.Resolver
	switch {
	case nodeOptions.Location.Country != "":
		locationResolver = location.NewResolverFake(nodeOptions.Location.Country)
	default:
		locationResolver = location.NewResolver(filepath.Join(nodeOptions.Directories.Config, nodeOptions.Location.Database))
	}

	return &Manager{
		networkDefinition: networkDefinition,
		identityLoader: func() (identity.Identity, error) {
			return identity_handler.LoadIdentity(identityHandler, serviceOptions.Identity, serviceOptions.Passphrase)
		},
		createSigner:     createSigner,
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		mysteriumClient:  mysteriumClient,
		natService:       natService,
		dialogWaiterFactory: func(myID identity.Identity, identityRegistry registry.IdentityRegistry) communication.DialogWaiter {
			return nats_dialog.NewDialogWaiter(
				nats_discovery.NewAddressGenerate(networkDefinition.BrokerAddress, myID),
				identity.NewSigner(keystoreInstance, myID),
				identityRegistry,
			)
		},

		sessionManagerFactory: func(primitives *tls.Primitives, vpnServerIP string) session.Manager {
			// TODO: check nodeOptions for --openvpn-transport option
			clientConfigGenerator := openvpn.NewClientConfigGenerator(
				primitives,
				vpnServerIP,
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)

			return session.NewManager(
				session.ServiceConfigProvider(clientConfigGenerator),
				&session.UUIDGenerator{},
			)
		},
		vpnServerFactory: func(manager session.Manager, primitives *tls.Primitives, callback state.Callback) openvpn.Process {
			// TODO: check nodeOptions for --openvpn-transport option
			serverConfigGenerator := openvpn.NewServerConfigGenerator(
				nodeOptions.Directories.Runtime,
				nodeOptions.Directories.Config,
				primitives,
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)

			ovpnSessionManager := openvpn_session.NewManager(manager)
			sessionValidator := openvpn_session.NewValidator(ovpnSessionManager, identity.NewExtractor())

			return openvpn.NewServer(
				nodeOptions.OpenvpnBinary,
				serverConfigGenerator,
				auth.NewMiddleware(sessionValidator.Validate, sessionValidator.Cleanup),
				state.NewMiddleware(callback),
			)
		},
		checkOpenvpn: func() error {
			return openvpn.CheckOpenvpnBinary(nodeOptions.OpenvpnBinary)
		},
		checkDirectories: nodeOptions.Directories.Check,
		openvpnServiceAddress: func(outboundIP, publicIP string) string {
			//TODO public ip could be overridden by arg nodeOptions if needed
			if publicIP != outboundIP {
				log.Warnf(
					`WARNING: It seems that publicly visible ip: [%s] does not match your local machines ip: [%s]. 
You should probably need to do port forwarding on your router: %s:%v -> %s:%v.`,
					publicIP,
					outboundIP,
					publicIP,
					serviceOptions.OpenvpnPort,
					outboundIP,
					serviceOptions.OpenvpnPort,
				)

			}

			return publicIP
		},
		protocol:                    serviceOptions.OpenvpnProtocol,
		proposalAnnouncementStopped: &sync.WaitGroup{},
	}
}
