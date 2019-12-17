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
	"net"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/bytecount"
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
	"github.com/mysteriumnetwork/node/session/event"
	"github.com/rs/zerolog/log"
)

const statisticsReportingIntervalInSeconds = 30

type eventBus interface {
	Publish(topic string, data interface{})
	SubscribeAsync(topic string, fn interface{}) error
}

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
	bus eventBus,
) *Manager {
	clientMap := openvpn_session.NewClientMap(sessionMap)

	sessionValidator := openvpn_session.NewValidator(clientMap, identity.NewExtractor())

	callback := func(sbc bytecount.SessionByteCount) {
		sessions := clientMap.GetClientSessions(sbc.ClientID)
		if len(sessions) == 1 {
			bus.Publish(event.DataTransfered, event.DataTransferEventPayload{
				ID:   string(sessions[0]),
				Up:   int64(sbc.BytesOut),
				Down: int64(sbc.BytesIn),
			})
		} else {
			log.Warn().Msgf("Could not map sessions - expected a single session to exist for a user, got %v sessions instead", len(sessions))
		}
	}

	return &Manager{
		publicIP:                       location.PubIP,
		outboundIP:                     location.OutIP,
		currentLocation:                location.Country,
		natService:                     natService,
		sessionConfigNegotiatorFactory: newSessionConfigNegotiatorFactory(nodeOptions.OptionsNetwork, serviceOptions, natEventGetter, portPool),
		vpnServerConfigFactory:         newServerConfigFactory(nodeOptions, serviceOptions),
		processLauncher:                newProcessLauncher(nodeOptions, sessionValidator, callback),
		natPingerPorts:                 port.NewPool(),
		natPinger:                      natPinger,
		serviceOptions:                 serviceOptions,
		mapPort:                        mapPort,
		natEventGetter:                 natEventGetter,
		ports:                          portPool,
		eventListener:                  bus,
	}
}

// newServerConfigFactory returns function generating server config and generates required security primitives
func newServerConfigFactory(nodeOptions node.Options, serviceOptions Options) ServerConfigFactory {
	return func(secPrimitives *tls.Primitives, port int) *openvpn_service.ServerConfig {
		return openvpn_service.NewServerConfig(
			nodeOptions.Directories.Runtime,
			nodeOptions.Directories.Config,
			serviceOptions.Subnet,
			serviceOptions.Netmask,
			secPrimitives,
			nodeOptions.BindAddress,
			port,
			serviceOptions.Protocol,
		)
	}
}

// newSessionConfigNegotiatorFactory returns function generating session config for remote client
func newSessionConfigNegotiatorFactory(networkOptions node.OptionsNetwork, serviceOptions Options, natEventGetter NATEventGetter, portPool port.ServicePortSupplier) SessionConfigNegotiatorFactory {
	return func(secPrimitives *tls.Primitives, dnsIP net.IP, outboundIP, publicIP string, port int) session.ConfigNegotiator {
		serverIP := vpnServerIP(serviceOptions, outboundIP, publicIP, networkOptions.Localnet)
		vpnConfig := &openvpn_service.VPNConfig{
			RemoteIP:        serverIP,
			RemotePort:      port,
			RemoteProtocol:  serviceOptions.Protocol,
			TLSPresharedKey: secPrimitives.PresharedKey.ToPEMFormat(),
			CACertificate:   secPrimitives.CertificateAuthority.ToPEMFormat(),
		}
		if dnsIP != nil {
			vpnConfig.DNSIPs = dnsIP.String()
		}
		return &OpenvpnConfigNegotiator{
			natEventGetter: natEventGetter,
			vpnConfig:      vpnConfig,
			portPool:       portPool,
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
func (ocn *OpenvpnConfigNegotiator) ProvideConfig(sessionConfig json.RawMessage, traversalParams *traversal.Params) (*session.ConfigParams, error) {
	ocn.vpnConfig.LocalPort = traversalParams.ConsumerPort
	ocn.vpnConfig.RemotePort = traversalParams.ProviderPort

	return &session.ConfigParams{SessionServiceConfig: ocn.vpnConfig, TraversalParams: traversalParams}, nil
}

func vpnServerIP(serviceOptions Options, outboundIP, publicIP string, isLocalnet bool) string {
	//TODO public ip could be overridden by arg nodeOptions if needed
	if publicIP == outboundIP {
		return publicIP
	}

	if isLocalnet {
		log.Warn().Msgf(
			`WARNING: It seems that publicly visible ip: [%s] does not match your local machines ip: [%s].
Since it's localnet, will use %v for openvpn service`, publicIP,
			outboundIP,
			outboundIP)
		return outboundIP
	}

	log.Warn().Msgf(
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
	log.Info().Msg("Country detected: " + currentCountry)

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
