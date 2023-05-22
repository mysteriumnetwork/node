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

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/filter"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/shaper"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/mysteriumnetwork/node/utils/stringutil"
)

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation market.Location) market.ServiceProposal

// Manager represents entrypoint for Openvpn service with top level components
type Manager struct {
	natService      nat.NATService
	ports           port.ServicePortSupplier
	dnsProxy        *dns.Proxy
	bus             eventbus.EventBus
	trafficFirewall firewall.IncomingTrafficFirewall
	vpnNetwork      net.IPNet
	vpnServerPort   int
	openvpnProcess  openvpn.Process
	openvpnClients  *clientMap
	openvpnAuth     *authHandler
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

	dnsPort := 11153
	dnsHandler, err := dns.ResolveViaSystem()
	if err == nil {
		if instance.PolicyProvider().HasDNSRules() {
			dnsHandler = dns.WhitelistAnswers(dnsHandler, m.trafficFirewall, instance.PolicyProvider())
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

	m.outboundIP, err = m.ipResolver.GetOutboundIP()
	if err != nil {
		return fmt.Errorf("could not get outbound IP: %w", err)
	}

	m.tlsPrimitives, err = primitiveFactory(m.country, instance.ProviderID.Address)
	if err != nil {
		return
	}

	if err := firewall.AddInboundRule(m.serviceOptions.Protocol, m.vpnServerPort); err != nil {
		return fmt.Errorf("failed to add firewall rule: %w", err)
	}
	defer func() {
		if err := firewall.RemoveInboundRule(m.serviceOptions.Protocol, m.vpnServerPort); err != nil {
			log.Error().Err(err).Msg("Failed to delete firewall rule for OpenVPN")
		}
	}()

	log.Info().Msgf("Starting OpenVPN server on port: %d", m.vpnServerPort)
	if err := m.startServer(); err != nil {
		return fmt.Errorf("failed to start Openvpn server: %w", err)
	}

	if _, err := m.natService.Setup(nat.Options{
		VPNNetwork:    m.vpnNetwork,
		ProviderExtIP: net.ParseIP(m.outboundIP),
		DNSIP:         m.dnsIP,
	}); err != nil {
		return fmt.Errorf("failed to setup NAT/firewall rules: %w", err)
	}

	s := shaper.New(m.bus)
	err = s.Start(m.openvpnProcess.DeviceName())
	if err != nil {
		log.Error().Err(err).Msg("Could not start traffic shaper")
	}
	defer s.Clear(m.openvpnProcess.DeviceName())

	log.Info().Msg("OpenVPN server waiting")
	return m.openvpnProcess.Wait()
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
func (m *Manager) ProvideConfig(sessionID string, sessionConfig json.RawMessage, conn *net.UDPConn) (*service.ConfigParams, error) {
	if m.vpnServerPort == 0 {
		return nil, errors.New("service port not initialized")
	}

	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, fmt.Errorf("could not get public IP: %w", err)
	}

	serverIP := vpnServerIP(m.outboundIP, publicIP, m.nodeOptions.OptionsNetwork.Network.IsLocalnet())
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

	if err := proxyOpenVPN(conn, m.vpnServerPort); err != nil {
		return nil, fmt.Errorf("could not proxy connection to OpenVPN server: %w", err)
	}

	destroy := func() {
		log.Info().Msgf("Cleaning up session %s", sessionID)

		sessionClients := m.openvpnClients.GetSessionClients(session.ID(sessionID))
		for clientID := range sessionClients {
			if err := m.openvpnAuth.ClientKill(clientID); err != nil {
				log.Error().Err(err).Msgf("Cleaning up session %s failed. Error disconnecting Openvpn client %d", sessionID, clientID)
			}
		}
	}

	return &service.ConfigParams{SessionServiceConfig: vpnConfig, SessionDestroyCallback: destroy}, nil
}

func (m *Manager) startServer() error {
	vpnServerConfig := NewServerConfig(
		m.nodeOptions.Directories.Runtime,
		m.nodeOptions.Directories.Script,
		m.serviceOptions.Subnet,
		m.serviceOptions.Netmask,
		m.tlsPrimitives,
		m.nodeOptions.BindAddress,
		m.vpnServerPort,
		m.serviceOptions.Protocol,
	)

	openvpnFilterDeny := stringutil.Split(config.GetString(config.FlagFirewallProtectedNetworks), ',')
	var openvpnFilterAllow []string
	if m.dnsOK {
		openvpnFilterAllow = []string{m.dnsIP.String()}
	}

	stateChannel := make(chan openvpn.State, 10)
	m.openvpnAuth = newAuthHandler(m.openvpnClients, identity.NewExtractor())
	m.openvpnProcess = openvpn.CreateNewProcess(
		m.nodeOptions.Openvpn.BinaryPath(),
		vpnServerConfig.GenericConfig,
		filter.NewMiddleware(openvpnFilterAllow, openvpnFilterDeny),
		m.openvpnAuth,
		state.NewMiddleware(func(state openvpn.State) {
			stateChannel <- state
			// this is the last state - close channel (according to best practices of go - channel writer controls channel)
			if state == openvpn.ProcessExited {
				close(stateChannel)
			}
		}),
		newStatsPublisher(m.openvpnClients, m.bus, 1),
	)
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
