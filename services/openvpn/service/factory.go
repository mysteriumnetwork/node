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
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
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
	sessionMap openvpn_session.SessionMap,
) *Manager {
	natService := nat.NewService()
	sessionValidator := openvpn_session.NewValidator(sessionMap, identity.NewExtractor())

	return &Manager{
		locationResolver:             locationResolver,
		ipResolver:                   ipResolver,
		natService:                   natService,
		proposalFactory:              newProposalFactory(serviceOptions),
		sessionConfigProviderFactory: newSessionConfigProviderFactory(serviceOptions),
		vpnServerConfigFactory:       newServerConfigFactory(nodeOptions, serviceOptions),
		vpnServerFactory:             newServerFactory(nodeOptions, sessionValidator),
	}
}

func newProposalFactory(serviceOptions service.Options) ProposalFactory {
	return func(currentLocation dto_discovery.Location) dto_discovery.ServiceProposal {
		return openvpn_discovery.NewServiceProposalWithLocation(currentLocation, serviceOptions.OpenvpnProtocol)
	}
}

// newServerConfigFactory returns function generating server config and generates required security primitives
func newServerConfigFactory(nodeOptions node.Options, serviceOptions service.Options) ServerConfigFactory {
	return func(secPrimitives *tls.Primitives) *openvpn_service.ServerConfig {
		// TODO: check nodeOptions for --openvpn-transport option
		return openvpn_service.NewServerConfig(
			nodeOptions.Directories.Runtime,
			nodeOptions.Directories.Config,
			"10.8.0.0", "255.255.255.0",
			secPrimitives,
			serviceOptions.OpenvpnPort,
			serviceOptions.OpenvpnProtocol,
		)
	}
}

func newServerFactory(nodeOptions node.Options, sessionValidator *openvpn_session.Validator) ServerFactory {
	return func(config *openvpn_service.ServerConfig) openvpn.Process {
		return openvpn.CreateNewProcess(
			nodeOptions.Openvpn.BinaryPath,
			config.GenericConfig,
			auth.NewMiddleware(sessionValidator.Validate, sessionValidator.Cleanup),
			state.NewMiddleware(vpnStateCallback),
		)
	}
}

func newSessionConfigProviderFactory(serviceOptions service.Options) SessionConfigProviderFactory {
	return func(secPrimitives *tls.Primitives, outboundIP, publicIP string) session.ConfigProvider {
		serverIP := vpnServerIP(serviceOptions, outboundIP, publicIP)

		return newSessionConfigProvider(serviceOptions, secPrimitives, serverIP)
	}
}

// newSessionConfigProvider returns function generating session config for remote client
func newSessionConfigProvider(serviceOptions service.Options, secPrimitives *tls.Primitives, serverIP string) session.ConfigProvider {
	// TODO: check nodeOptions for --openvpn-transport option
	return func() (session.ServiceConfiguration, error) {
		return &openvpn_service.VPNConfig{
			serverIP,
			serviceOptions.OpenvpnPort,
			serviceOptions.OpenvpnProtocol,
			secPrimitives.PresharedKey.ToPEMFormat(),
			secPrimitives.CertificateAuthority.ToPEMFormat(),
		}, nil
	}
}

func vpnServerIP(serviceOptions service.Options, outboundIP, publicIP string) string {
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
