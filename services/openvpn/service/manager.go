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
	"net"
	"strings"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/tls"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/shaper"
	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/mapping"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/services"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ServerConfigFactory callback generates session config for remote client
type ServerConfigFactory func(secPrimitives *tls.Primitives, port int) *openvpn_service.ServerConfig

// ProposalFactory prepares service proposal during runtime
type ProposalFactory func(currentLocation market.Location) market.ServiceProposal

// SessionConfigNegotiatorFactory initiates ConfigProvider instance during runtime
type SessionConfigNegotiatorFactory func(secPrimitives *tls.Primitives, dnsIP net.IP, outboundIP, publicIP string, port int) session.ConfigNegotiator

// NATPinger defined Pinger interface for Provider
type NATPinger interface {
	BindServicePort(serviceType services.ServiceType, port int)
	Stop()
	Valid() bool
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
	natService     nat.NATService
	mapPort        func(int) (releasePortMapping func())
	ports          port.ServicePortSupplier
	natPingerPorts port.ServicePortSupplier
	natPinger      NATPinger
	natEventGetter NATEventGetter
	dnsProxy       *dns.Proxy
	eventListener  eventListener

	sessionConfigNegotiatorFactory SessionConfigNegotiatorFactory
	consumerConfig                 openvpn_service.ConsumerConfig

	vpnNetwork               net.IPNet
	vpnServerPort            int
	vpnServerConfigFactory   ServerConfigFactory
	vpnServiceConfigProvider session.ConfigNegotiator
	processLauncher          *processLauncher
	openvpnProcess           openvpn.Process

	publicIP        string
	outboundIP      string
	currentLocation string
	serviceOptions  Options
}

// Serve starts service - does block
func (m *Manager) Serve(providerID identity.Identity) (err error) {
	m.vpnNetwork = net.IPNet{
		IP:   net.ParseIP(m.serviceOptions.Subnet),
		Mask: net.IPMask(net.ParseIP(m.serviceOptions.Netmask).To4()),
	}

	var dnsOK bool
	var dnsIP net.IP
	var dnsPort = 11153
	m.dnsProxy = dns.NewProxy("", dnsPort)
	if err := m.dnsProxy.Run(); err != nil {
		log.Warn().Err(err).Msg("Provider DNS will not be available")
	} else {
		dnsOK = true
		dnsIP = netutil.FirstIP(m.vpnNetwork)
	}

	servicePort, err := m.ports.Acquire()
	if err != nil {
		return errors.Wrap(err, "failed to acquire an unused port")
	}
	m.vpnServerPort = servicePort.Num()

	releasePorts := m.mapPort(m.vpnServerPort)
	defer releasePorts()

	primitives, err := primitiveFactory(m.currentLocation, providerID.Address)
	if err != nil {
		return
	}

	m.vpnServiceConfigProvider = m.sessionConfigNegotiatorFactory(primitives, dnsIP, m.outboundIP, m.publicIP, m.vpnServerPort)

	vpnServerConfig := m.vpnServerConfigFactory(primitives, m.vpnServerPort)
	stateChannel := make(chan openvpn.State, 10)

	protectedNetworks := strings.FieldsFunc(config.GetString(config.FlagFirewallProtectedNetworks), func(c rune) bool { return c == ',' })
	var openvpnFilterAllow []string
	if dnsOK {
		openvpnFilterAllow = []string{dnsIP.String()}
	}
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
		return errors.Wrap(err, "failed to add firewall rule")
	}
	defer func() {
		if err := firewall.RemoveInboundRule(m.serviceOptions.Protocol, m.vpnServerPort); err != nil {
			log.Error().Err(err).Msg("Failed to delete firewall rule for OpenVPN")
		}
	}()

	if err := m.startServer(m.openvpnProcess, stateChannel); err != nil {
		return errors.Wrap(err, "failed to start Openvpn server")
	}

	if _, err := m.natService.Setup(nat.Options{
		VPNNetwork:        m.vpnNetwork,
		ProviderExtIP:     net.ParseIP(m.outboundIP),
		EnableDNSRedirect: dnsOK,
		DNSIP:             dnsIP,
		DNSPort:           dnsPort,
	}); err != nil {
		return errors.Wrap(err, "failed to setup NAT/firewall rules")
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

// Stop stops service
func (m *Manager) Stop() error {
	if m.openvpnProcess != nil {
		m.openvpnProcess.Stop()
	}

	errStop := utils.ErrorCollection{}
	if m.dnsProxy != nil {
		errStop.Add(m.dnsProxy.Stop())
	}

	return errStop.Errorf("ErrorCollection(%s)", ", ")
}

// ProvideConfig takes session creation config from end consumer and provides the service configuration to the end consumer
func (m *Manager) ProvideConfig(sessionConfig json.RawMessage, traversalParams *traversal.Params) (*session.ConfigParams, error) {
	if m.vpnServiceConfigProvider == nil {
		return nil, errors.New("Config provider not initialized")
	}
	if m.vpnServerPort == 0 {
		return nil, errors.New("Service port not initialized")
	}

	traversalParams = &traversal.Params{ProviderPort: m.vpnServerPort}

	// Older clients do not send any sessionConfig, but we should keep back compatibility and not fail in this case.
	if sessionConfig != nil && len(sessionConfig) > 0 && m.natPinger.Valid() {
		var c openvpn_service.ConsumerConfig
		err := json.Unmarshal(sessionConfig, &c)
		if err != nil {
			return nil, errors.Wrap(err, "parsing consumer sessionConfig failed")
		}
		m.consumerConfig = c

		if m.isBehindNAT() && m.portMappingFailed() {
			pp, err := m.natPingerPorts.Acquire()
			if err != nil {
				return nil, err
			}

			cp, err := m.natPingerPorts.Acquire()
			if err != nil {
				return nil, err
			}

			traversalParams.ProviderPort = pp.Num()
			traversalParams.ConsumerPort = cp.Num()
		}
	}

	return m.vpnServiceConfigProvider.ProvideConfig(sessionConfig, traversalParams)
}

func (m *Manager) startServer(server openvpn.Process, stateChannel chan openvpn.State) error {
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

func (m *Manager) isBehindNAT() bool {
	return m.outboundIP != m.publicIP
}

func (m *Manager) portMappingFailed() bool {
	event := m.natEventGetter.LastEvent()
	if event == nil {
		return false
	}

	if event.Stage == traversal.StageName {
		return true
	}
	return event.Stage == mapping.StageName && !event.Successful
}
