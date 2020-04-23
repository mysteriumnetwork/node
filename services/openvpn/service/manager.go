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
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/shaper"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/mysteriumnetwork/node/utils/stringutil"
	"github.com/rs/zerolog/log"
)

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation market.Location) market.ServiceProposal

type natPinger interface {
	BindServicePort(key string, port int)
	Stop()
}

// NATEventGetter allows us to fetch the last known NAT event
type NATEventGetter interface {
	LastEvent() *event.Event
}

type eventListener interface {
	SubscribeAsync(topic string, fn interface{}) error
}

// Manager represents entrypoint for Openvpn service with top level components
type Manager struct {
	natService      nat.NATService
	ports           port.ServicePortSupplier
	natPingerPorts  port.ServicePortSupplier
	natPinger       natPinger
	natEventGetter  NATEventGetter
	dnsProxy        *dns.Proxy
	eventListener   eventListener
	portMapper      mapping.PortMapper
	trafficFirewall firewall.IncomingTrafficFirewall
	vpnNetwork      net.IPNet
	vpnServerPort   int
	processLauncher *processLauncher
	openvpnProcess  openvpn.Process
	ipResolver      ip.Resolver
	serviceOptions  Options
	nodeOptions     node.Options

	outboundIP    string
	country       string
	dnsIP         net.IP
	dnsOK         bool
	tlsPrimitives *tls.Primitives
}

// Serve starts service - does block
func (m *Manager) Serve(instance *service.Instance) (err error) {
	m.vpnNetwork = net.IPNet{
		IP:   net.ParseIP(m.serviceOptions.Subnet),
		Mask: net.IPMask(net.ParseIP(m.serviceOptions.Netmask).To4()),
	}

	var dnsPort = 11153
	dnsHandler, err := dns.ResolveViaSystem()
	if err == nil {
		if instance.Policies().HasDNSRules() {
			dnsHandler = dns.WhitelistAnswers(dnsHandler, m.trafficFirewall, instance.Policies())
			removeRule, err := m.trafficFirewall.BlockIncomingTraffic(m.vpnNetwork)
			if err != nil {
				return fmt.Errorf("failed to enable traffic blocking: %w", err)
			}
			defer func() {
				if err := removeRule(); err != nil {
					log.Warn().Err(err).Msg("failed to disable traffic blocking")
				}
			}()
		}

		m.dnsProxy = dns.NewProxy("", dnsPort, dnsHandler)
		if err := m.dnsProxy.Run(); err != nil {
			log.Warn().Err(err).Msg("Provider DNS will not be available")
		} else {
			m.dnsOK = true
			m.dnsIP = netutil.FirstIP(m.vpnNetwork)
		}
	} else {
		log.Warn().Err(err).Msg("Provider DNS will not be available")
	}

	servicePort, err := m.ports.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire an unused port: %w", err)
	}
	m.vpnServerPort = servicePort.Num()

	m.outboundIP, err = m.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return fmt.Errorf("could not get outbound IP: %w", err)
	}

	pubIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return fmt.Errorf("could not get public IP: %w", err)
	}

	if m.behindNAT(pubIP) {
		if releasePorts, ok := m.tryAddPortMapping(m.vpnServerPort); ok {
			defer releasePorts()
		}
	}

	m.tlsPrimitives, err = primitiveFactory(m.country, instance.Proposal().ProviderID)
	if err != nil {
		return
	}

	stateChannel := make(chan openvpn.State, 10)

	protectedNetworks := stringutil.Split(config.GetString(config.FlagFirewallProtectedNetworks), ',')
	var openvpnFilterAllow []string
	if m.dnsOK {
		openvpnFilterAllow = []string{m.dnsIP.String()}
	}

	vpnServerConfig := NewServerConfig(
		m.nodeOptions.Directories.Runtime,
		m.nodeOptions.Directories.Config,
		m.serviceOptions.Subnet,
		m.serviceOptions.Netmask,
		m.tlsPrimitives,
		m.nodeOptions.BindAddress,
		m.vpnServerPort,
		m.serviceOptions.Protocol,
	)

	m.openvpnProcess = m.processLauncher.launch(launchOpts{
		config:       vpnServerConfig,
		filterAllow:  openvpnFilterAllow,
		filterBlock:  protectedNetworks,
		stateChannel: stateChannel,
	})

	// register service port to which NATProxy will forward connects attempts to
	m.natPinger.BindServicePort(openvpn_service.ServiceType, m.vpnServerPort)

	log.Info().Msgf("Starting OpenVPN server on port: %d", m.vpnServerPort)
	if err := firewall.AddInboundRule(m.serviceOptions.Protocol, m.vpnServerPort); err != nil {
		return fmt.Errorf("failed to add firewall rule: %w", err)
	}
	defer func() {
		if err := firewall.RemoveInboundRule(m.serviceOptions.Protocol, m.vpnServerPort); err != nil {
			log.Error().Err(err).Msg("Failed to delete firewall rule for OpenVPN")
		}
	}()

	if err := m.startServer(stateChannel); err != nil {
		return fmt.Errorf("failed to start Openvpn server: %w", err)
	}

	if _, err := m.natService.Setup(nat.Options{
		VPNNetwork:        m.vpnNetwork,
		ProviderExtIP:     net.ParseIP(m.outboundIP),
		EnableDNSRedirect: m.dnsOK,
		DNSIP:             m.dnsIP,
		DNSPort:           dnsPort,
	}); err != nil {
		return fmt.Errorf("failed to setup NAT/firewall rules: %w", err)
	}

	s := shaper.New(m.eventListener)
	err = s.Start(m.openvpnProcess.DeviceName())
	if err != nil {
		log.Error().Err(err).Msg("Could not start traffic shaper")
	}
	defer s.Clear(m.openvpnProcess.DeviceName())

	log.Info().Msg("OpenVPN server waiting")
	return m.openvpnProcess.Wait()
}

func (m *Manager) tryAddPortMapping(port int) (release func(), ok bool) {
	release, ok = m.portMapper.Map(
		m.serviceOptions.Protocol,
		port,
		"Myst node OpenVPN port mapping")

	return release, ok
}

// Stop stops service
func (m *Manager) Stop() error {
	if m.openvpnProcess != nil {
		m.openvpnProcess.Stop()
	}

	if m.dnsProxy != nil {
		if err := m.dnsProxy.Stop(); err != nil {
			return fmt.Errorf("could not stop DNS proxy: %w", err)
		}
	}

	return nil
}

// ProvideConfig takes session creation config from end consumer and provides the service configuration to the end consumer
func (m *Manager) ProvideConfig(_ string, sessionConfig json.RawMessage, conn *net.UDPConn) (*session.ConfigParams, error) {
	if m.vpnServerPort == 0 {
		return nil, errors.New("service port not initialized")
	}

	traversalParams := traversal.Params{}

	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, fmt.Errorf("could not get public IP: %w", err)
	}

	serverIP := vpnServerIP(m.serviceOptions.Port, m.outboundIP, publicIP, m.nodeOptions.OptionsNetwork.Localnet)
	vpnConfig := &openvpn_service.VPNConfig{
		RemoteIP:        serverIP,
		RemotePort:      m.vpnServerPort,
		RemoteProtocol:  m.serviceOptions.Protocol,
		TLSPresharedKey: m.tlsPrimitives.PresharedKey.ToPEMFormat(),
		CACertificate:   m.tlsPrimitives.CertificateAuthority.ToPEMFormat(),
	}
	if m.dnsOK {
		vpnConfig.DNSIPs = m.dnsIP.String()
	}

	if conn == nil { // TODO this backward compatibility block needs to be removed once we will fully migrate to the p2p communication.
		if _, noop := m.natPinger.(*traversal.NoopPinger); noop {
			return &session.ConfigParams{SessionServiceConfig: vpnConfig}, nil
		}

		var consumerConfig openvpn_service.ConsumerConfig
		err := json.Unmarshal(sessionConfig, &consumerConfig)
		if err != nil {
			return nil, fmt.Errorf("could not parse consumer config: %w", err)
		}

		if m.behindNAT(publicIP) && m.portMappingFailed() {
			for range consumerConfig.Ports {
				pp, err := m.natPingerPorts.Acquire()
				if err != nil {
					return nil, err
				}

				vpnConfig.Ports = append(vpnConfig.Ports, pp.Num())
				vpnConfig.RemotePort = pp.Num()
			}

			// For OpenVPN only one running NAT proxy required.
			if consumerConfig.IP == "" {
				return nil, errors.New("remote party does not support NAT Hole punching, public IP is missing")
			}

			traversalParams.IP = consumerConfig.IP
			traversalParams.LocalPorts = vpnConfig.Ports
			traversalParams.RemotePorts = consumerConfig.Ports
			traversalParams.ProxyPortMappingKey = openvpn_service.ServiceType
		}
	} else {
		if err := proxyOpenVPN(conn, m.vpnServerPort); err != nil {
			return nil, fmt.Errorf("could not proxy connection to OpenVPN server: %w", err)
		}
	}
	return &session.ConfigParams{SessionServiceConfig: vpnConfig, TraversalParams: traversalParams}, nil
}

func (m *Manager) startServer(stateChannel chan openvpn.State) error {
	if err := m.openvpnProcess.Start(); err != nil {
		return err
	}

	// Wait for started state
	for {
		state, more := <-stateChannel
		if !more {
			return errors.New("process failed to start")
		}
		if state == openvpn.ConnectedState {
			break
		}
	}

	// Consume server states
	go func() {
		for state := range stateChannel {
			switch state {
			case openvpn.ProcessStarted:
				log.Info().Msg("OpenVPN service booting up")
			case openvpn.ProcessExited:
				log.Info().Msg("OpenVPN service exited")
			}
		}
	}()

	log.Info().Msg("OpenVPN service started successfully")
	return nil
}

func (m *Manager) portMappingFailed() bool {
	lastEvent := m.natEventGetter.LastEvent()
	if lastEvent == nil {
		return false
	}

	if lastEvent.Stage == traversal.StageName {
		return true
	}
	return lastEvent.Stage == mapping.StageName && !lastEvent.Successful
}

func (m *Manager) behindNAT(pubIP string) bool {
	return m.outboundIP != pubIP
}
