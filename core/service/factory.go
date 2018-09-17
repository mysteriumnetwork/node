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
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	nats_dialog "github.com/mysteriumnetwork/node/communication/nats/dialog"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/discovery"
	"github.com/mysteriumnetwork/node/identity"
	identity_registry "github.com/mysteriumnetwork/node/identity/registry"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/openvpn"
	"github.com/mysteriumnetwork/node/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/node/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/openvpn/tls"
	openvpn_node "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/mysteriumnetwork/node/session"
)

// NewManager function creates new service manager by given options
func NewManager(
	nodeOptions node.Options,
	serviceOptions Options,
	networkDefinition metadata.NetworkDefinition,
	identityLoader identity_selector.Loader,
	signerFactory identity.SignerFactory,
	identityRegistry identity_registry.IdentityRegistry,
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
	discoveryService *discovery.Discovery,
) *Manager {
	logconfig.Bootstrap()

	natService := nat.NewService()

	return &Manager{
		identityLoader:   identityLoader,
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		natService:       natService,
		dialogWaiterFactory: func(myID identity.Identity) communication.DialogWaiter {
			return nats_dialog.NewDialogWaiter(
				nats_discovery.NewAddressGenerate(networkDefinition.BrokerAddress, myID),
				signerFactory(myID),
				identityRegistry,
			)
		},

		sessionManagerFactory: func(primitives *tls.Primitives, vpnServerIP string) session.Manager {
			// TODO: check nodeOptions for --openvpn-transport option
			clientConfigGenerator := openvpn_node.NewClientConfigGenerator(
				primitives,
				vpnServerIP,
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)
			serviceConfigProvider := func() (session.ServiceConfiguration, error) {
				return clientConfigGenerator(), nil
			}
			return session.NewManager(
				serviceConfigProvider,
				&session.UUIDGenerator{},
			)
		},
		vpnServerFactory: func(manager session.Manager, primitives *tls.Primitives, callback state.Callback) openvpn.Process {
			// TODO: check nodeOptions for --openvpn-transport option
			serverConfigGenerator := openvpn_node.NewServerConfigGenerator(
				nodeOptions.Directories.Runtime,
				nodeOptions.Directories.Config,
				primitives,
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)

			ovpnSessionManager := openvpn_session.NewManager(manager)
			sessionValidator := openvpn_session.NewValidator(ovpnSessionManager, identity.NewExtractor())

			return openvpn_node.NewServer(
				nodeOptions.Openvpn.BinaryPath,
				serverConfigGenerator,
				auth.NewMiddleware(sessionValidator.Validate, sessionValidator.Cleanup),
				state.NewMiddleware(callback),
			)
		},
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
		protocol:  serviceOptions.OpenvpnProtocol,
		discovery: discoveryService,
	}
}
