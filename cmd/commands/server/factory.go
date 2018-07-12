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

package server

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/cmd"
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
	"github.com/mysterium/node/openvpn/tls"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/session"
	"path/filepath"
	"sync"
)

// NewCommand function creates new server command by given options
func NewCommand(options CommandOptions) *Command {

	networkDefinition := cmd.GetNetworkDefinition(options.NetworkOptions)

	mysteriumClient := server.NewClient(networkDefinition.DiscoveryAPIAddress)
	ipResolver := ip.NewResolver(options.IpifyUrl)
	natService := nat.NewService()

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

	locationResolver := locationResolver(options)

	return &Command{
		identityLoader: func() (identity.Identity, error) {
			return identity_handler.LoadIdentity(identityHandler, options.Identity, options.Passphrase)
		},
		createSigner:     createSigner,
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		mysteriumClient:  mysteriumClient,
		natService:       natService,
		dialogWaiterFactory: func(myID identity.Identity) communication.DialogWaiter {
			return nats_dialog.NewDialogWaiter(
				nats_discovery.NewAddressGenerate(networkDefinition.BrokerAddress, myID),
				identity.NewSigner(keystoreInstance, myID),
			)
		},

		sessionManagerFactory: func(primitives *tls.Primitives, vpnServerIP string) session.Manager {
			// TODO: check options for --openvpn-transport option
			clientConfigGenerator := openvpn.NewClientConfigGenerator(
				primitives,
				vpnServerIP,
				options.OpenvpnPort,
				options.Protocol,
			)

			return session.NewManager(
				session.ServiceConfigProvider(clientConfigGenerator),
				&session.UUIDGenerator{},
			)
		},
		vpnServerFactory: func(manager session.Manager, primitives *tls.Primitives, callback state.Callback) openvpn.Process {
			// TODO: check options for --openvpn-transport option
			serverConfigGenerator := openvpn.NewServerConfigGenerator(
				options.DirectoryRuntime,
				options.DirectoryConfig,
				primitives,
				options.OpenvpnPort,
				options.Protocol,
			)

			ovpnSessionManager := openvpn_session.NewManager(manager)
			sessionValidator := openvpn_session.NewValidator(ovpnSessionManager, identity.NewExtractor())

			return openvpn.NewServer(
				options.OpenvpnBinary,
				serverConfigGenerator,
				auth.NewMiddleware(sessionValidator.Validate, sessionValidator.Cleanup),
				state.NewMiddleware(callback),
			)
		},
		checkOpenvpn: func() error {
			return openvpn.CheckOpenvpnBinary(options.OpenvpnBinary)
		},
		openvpnServiceAddress: func(outboundIP, publicIP string) string {
			//TODO public ip could be overriden by arg options if needed
			if publicIP != outboundIP {
				forwardInfo := fmt.Sprintf("%s:%v -> %s:%v", publicIP, options.OpenvpnPort, outboundIP, options.OpenvpnPort)
				log.Warnf(
					`WARNING: It seems that publicaly visible ip: [%s] does not match your local machines ip: [%s]. 
You should probaly need to do port forwarding on your router: %s.`,
					publicIP,
					outboundIP,
					forwardInfo,
				)

			}

			return publicIP
		},
		protocol:                    options.Protocol,
		proposalAnnouncementStopped: &sync.WaitGroup{},
	}
}

func locationResolver(options CommandOptions) location.Resolver {
	switch {
	case options.LocationCountry != "":
		return location.NewResolverFake(options.LocationCountry)
	default:
		return location.NewResolver(filepath.Join(options.DirectoryConfig, options.LocationDatabase))
	}
}
