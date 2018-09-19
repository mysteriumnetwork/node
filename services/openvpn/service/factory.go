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
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	dto_discovery "github.com/mysteriumnetwork/node/service_discovery/dto"
	openvpn_node "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_discovery "github.com/mysteriumnetwork/node/services/openvpn/discovery"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/mysteriumnetwork/node/session"
)

// NewManager creates new instance of Openvpn service
func NewManager(
	nodeOptions node.Options,
	serviceOptions service.Options,
	ipResolver ip.Resolver,
	locationResolver location.Resolver,
) *Manager {
	natService := nat.NewService()
	sessionStorage := session.NewStorageMemory()

	vpnServerIP := func(outboundIP, publicIP string) string {
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
	}

	return &Manager{
		locationResolver: locationResolver,
		ipResolver:       ipResolver,
		natService:       natService,
		proposalFactory: func(currentLocation dto_discovery.Location) dto_discovery.ServiceProposal {
			return openvpn_discovery.NewServiceProposalWithLocation(currentLocation, serviceOptions.OpenvpnProtocol)
		},
		sessionManagerFactory: func(primitives *tls.Primitives, outboundIP, publicIP string) session.Manager {
			// TODO: check nodeOptions for --openvpn-transport option
			clientConfigGenerator := openvpn_node.NewClientConfigGenerator(
				primitives,
				vpnServerIP(outboundIP, publicIP),
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)
			serviceConfigProvider := func() (session.ServiceConfiguration, error) {
				return clientConfigGenerator(), nil
			}
			return session.NewManager(session.GenerateUUID, serviceConfigProvider, sessionStorage.Add)
		},
		vpnServerFactory: func(primitives *tls.Primitives) openvpn.Process {
			// TODO: check nodeOptions for --openvpn-transport option
			serverConfigGenerator := openvpn_node.NewServerConfigGenerator(
				nodeOptions.Directories.Runtime,
				nodeOptions.Directories.Config,
				primitives,
				serviceOptions.OpenvpnPort,
				serviceOptions.OpenvpnProtocol,
			)

			sessionValidator := openvpn_session.NewValidator(sessionStorage, identity.NewExtractor())

			return openvpn_node.NewServer(
				nodeOptions.Openvpn.BinaryPath,
				serverConfigGenerator,
				auth.NewMiddleware(sessionValidator.Validate, sessionValidator.Cleanup),
				state.NewMiddleware(vpnStateCallback),
			)
		},
	}
}
