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
	"crypto/x509/pkix"
	"encoding/json"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/traversal"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
	"github.com/mysteriumnetwork/node/session"
)

// NewManager creates new instance of Openvpn service
func NewManager(nodeOptions node.Options,
	serviceOptions Options,
	location location.ServiceLocationInfo,
	sessionMap openvpn_session.SessionMap,
	natService nat.NATService,
	natPinger NATPinger,
	mapPort func(int) (releasePortMapping func()),
	natEventGetter NATEventGetter,
	portPool port.ServicePortSupplier,
) *Manager {
	sessionValidator := openvpn_session.NewValidator(sessionMap, identity.NewExtractor())

	serverFactory := newServerFactory(nodeOptions, sessionValidator)

	return &Manager{
		publicIP:                       location.PubIP,
		outboundIP:                     location.OutIP,
		currentLocation:                location.Country,
		natService:                     natService,
		sessionConfigNegotiatorFactory: newSessionConfigNegotiatorFactory(nodeOptions.OptionsNetwork, serviceOptions, natEventGetter, portPool),
		vpnServerConfigFactory:         newServerConfigFactory(nodeOptions, serviceOptions),
		vpnServerFactory:               serverFactory,
		natPinger:                      natPinger,
		serviceOptions:                 serviceOptions,
		mapPort:                        mapPort,
		natEventGetter:                 natEventGetter,
		ports:                          portPool,
	}
}

// newServerConfigFactory returns function generating server config and generates required security primitives
func newServerConfigFactory(nodeOptions node.Options, serviceOptions Options) ServerConfigFactory {
	return func(secPrimitives *tls.Primitives, port int) *openvpn_service.ServerConfig {
		return openvpn_service.NewServerConfig(
			nodeOptions.Directories.Runtime,
			nodeOptions.Directories.Config,
			"10.8.0.0", "255.255.255.0",
			secPrimitives,
			port,
			serviceOptions.Protocol,
		)
	}
}

func newServerFactory(nodeOptions node.Options, sessionValidator *openvpn_session.Validator) ServerFactory {
	return func(config *openvpn_service.ServerConfig) openvpn.Process {
		return openvpn.CreateNewProcess(
			nodeOptions.Openvpn.BinaryPath(),
			config.GenericConfig,
			auth.NewMiddleware(sessionValidator.Validate),
			state.NewMiddleware(vpnStateCallback),
		)
	}
}

// newSessionConfigNegotiatorFactory returns function generating session config for remote client
func newSessionConfigNegotiatorFactory(networkOptions node.OptionsNetwork, serviceOptions Options, natEventGetter NATEventGetter, portPool port.ServicePortSupplier) SessionConfigNegotiatorFactory {
	return func(secPrimitives *tls.Primitives, outboundIP, publicIP string, port int) session.ConfigNegotiator {
		serverIP := vpnServerIP(serviceOptions, outboundIP, publicIP, networkOptions.Localnet)
		return &OpenvpnConfigNegotiator{
			natEventGetter: natEventGetter,
			vpnConfig: &openvpn_service.VPNConfig{
				RemoteIP:        serverIP,
				RemotePort:      port,
				RemoteProtocol:  serviceOptions.Protocol,
				TLSPresharedKey: secPrimitives.PresharedKey.ToPEMFormat(),
				CACertificate:   secPrimitives.CertificateAuthority.ToPEMFormat(),
			},
			portPool: portPool,
		}
	}
}

// OpenvpnConfigNegotiator knows how to send the openvpn config to the consumer
type OpenvpnConfigNegotiator struct {
	natEventGetter NATEventGetter
	vpnConfig      *openvpn_service.VPNConfig
	portPool       port.ServicePortSupplier
}

// ProvideConfig returns the config for user
func (ocn *OpenvpnConfigNegotiator) ProvideConfig(consumerKey json.RawMessage, params *traversal.Params) (session.ServiceConfiguration, session.DestroyCallback, *traversal.Params, error) {
	ocn.vpnConfig.LocalPort = params.ConsumerPort
	ocn.vpnConfig.RemotePort = params.ProviderPort

	return ocn.vpnConfig, nil, params, nil
}

func vpnServerIP(serviceOptions Options, outboundIP, publicIP string, isLocalnet bool) string {
	//TODO public ip could be overridden by arg nodeOptions if needed
	if publicIP == outboundIP {
		return publicIP
	}

	if isLocalnet {
		log.Warnf(
			`WARNING: It seems that publicly visible ip: [%s] does not match your local machines ip: [%s].
Since it's localnet, will use %v for openvpn service`, publicIP,
			outboundIP,
			outboundIP)
		return outboundIP
	}

	log.Warnf(
		`WARNING: It seems that publicly visible ip: [%s] does not match your local machines ip: [%s].
You should probably need to do port forwarding on your router: %s:%v -> %s:%v.`,
		publicIP,
		outboundIP,
		publicIP,
		serviceOptions.Port,
		outboundIP,
		serviceOptions.Port,
	)
	return publicIP
}

// primitiveFactory takes in the country and providerID and forms the tls primitives out of it
func primitiveFactory(currentCountry, providerID string) (*tls.Primitives, error) {
	log.Info(logPrefix, "Country detected: ", currentCountry)

	caSubject := pkix.Name{
		Country:            []string{currentCountry},
		Organization:       []string{"Mysterium Network"},
		OrganizationalUnit: []string{"Mysterium Team"},
	}
	serverCertSubject := pkix.Name{
		Country:            []string{currentCountry},
		Organization:       []string{"Mysterium node operator company"},
		OrganizationalUnit: []string{"Node operator team"},
		CommonName:         providerID,
	}

	return tls.NewTLSPrimitives(caSubject, serverCertSubject)
}
